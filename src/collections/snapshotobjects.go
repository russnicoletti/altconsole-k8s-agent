package collections

import (
	"altc-agent/altc"
	altcinformers "altc-agent/informers"
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/util/workqueue"
	"time"
)

const _snapshotObjectsQName = "altc-snapshotObjectsQ"

type SnapshotObjectsContext struct {
	BatchLimit              int
	SnapshotIntervalSeconds int
	ClusterName             string
}

type SnapshotObjects struct {
	SnapshotObjectsContext // Expose externally in order to log configuration

	queue           workqueue.Interface
	resourceObjects *ResourceObjects
	informers       []*altcinformers.Informer
	client          *altc.Client
}

func NewSnapshotObjects(resourceObjects *ResourceObjects, informers []*altcinformers.Informer, context SnapshotObjectsContext) *SnapshotObjects {
	queue := workqueue.NewNamed(_snapshotObjectsQName)

	return &SnapshotObjects{
		SnapshotObjectsContext: context,
		queue:                  queue,
		resourceObjects:        resourceObjects,
		informers:              informers,
		client:                 altc.NewClient(),
	}
}

func (so *SnapshotObjects) Loop(ctx context.Context) {

	for {
		ready, stop := scheduleCollection(so.collectResourceObjects, time.Duration(so.SnapshotObjectsContext.SnapshotIntervalSeconds)*time.Second, ctx.Done())
		select {
		case <-ready:
			fmt.Println("Number of cluster objects:", so.resourceObjects.Count())
			snapshotId := uuid.NewUUID()
			fmt.Println("snapshotId:", snapshotId)

			// Process all resource objects in the snapshot, taking into account
			// batch size
			for ok := true; ok; ok = so.resourceObjects.Count() != 0 {
				so.populate(snapshotId)
				snapshotObject, shutdown := so.getSnapshotObject()

				// TODO Add error handling on shutdown (controller needs to react)
				if shutdown {
					fmt.Println(fmt.Sprintf("%T shutdown", SnapshotObjects{}))
					return
				}

				itemsToSend := len(snapshotObject.Data)
				err := so.client.Send(ctx, snapshotObject)

				// Ack the snapshotObjects queue item regardless of whether the item
				// was successfully sent to the server.
				//
				//  If the item was sent to the server:
				//   The item needs to be acked to indicate the queue item is finished being
				//   processed (the presence of items on the queue that are not finished being
				//   processed will prevent the queue from being shutdown).
				//
				//  If the item was not successfully sent to the server:
				//   The semantics of adding an item to a workqueue is such that the item won't be re-added if it
				//   is still "processing". Therefore, the item needs to be acked before being
				//   re-added.
				//
				so.queue.Done(snapshotObject)

				if err != nil {
					fmt.Println("ERROR: error sending resources to server:", err)
					so.queue.Add(snapshotObject)
					continue
				}
				fmt.Println(fmt.Sprintf("after sending %d items, resourceObjects len: %d",
					itemsToSend, so.resourceObjects.Count()))
			}
			break
		case <-stop:
			fmt.Println("collection of resources has been stopped")
			return
		}
	}
}

func scheduleCollection(collect func(), delay time.Duration, done <-chan struct{}) (<-chan bool, <-chan bool) {
	fmt.Println("scheduling snapshot object collection for", time.Now().Add(delay))

	ready := make(chan bool)
	stop := make(chan bool)
	go func() {
		for {
			select {
			case <-time.After(delay):
				collect()
				ready <- true
				return
			case <-done:
				fmt.Println("scheduleCollection is stopped")
				stop <- true
				return
			}
		}
	}()
	return ready, stop
}

func (so *SnapshotObjects) collectResourceObjects() {
	// Shouldn't be collecting objects until all previously collected
	// objects have been sent to the server
	if so.queue.Len() != 0 {
		fmt.Println("snapshotObjectsQ is not empty, not adding additional resources")
		return
	}

	fmt.Println("collecting snapshot objects at", time.Now())
	for _, informer := range so.informers {
		resourcesList := informer.Informer.GetStore().List()
		fmt.Println(fmt.Sprintf("collecting %d objects from %s informer", len(resourcesList), informer.Name))
		for _, item := range resourcesList {
			so.addResourceObject(item)
		}
	}
	fmt.Println("finished collecting objects at", time.Now())
}

func (so *SnapshotObjects) addResourceObject(obj interface{}) {
	resourceObject, ok := obj.(altc.ResourceObject)
	if !ok {
		fmt.Println("WARN: 'obj' is not an altc.ResourceObject")
		return
	}

	// Remove managed fields, if present
	if metadata, ok := obj.(metav1.Object); !ok {
		metadata.SetManagedFields(nil)
	}

	// TODO Is 'Action' useful? It is a relic of the initial implementation that used
	// the event-driven model where the informers would send 'add/update/delete' events...
	if err := so.resourceObjects.AddItem("todo-remove-action-from-schema", resourceObject); err != nil {
		fmt.Println(err.Error())
	}
}

// populate
//
// Add items to the snapshot objects queue, respecting
// the batch size. If the queue is already populated,
// does not add any additional resources.
func (so *SnapshotObjects) populate(snapshotId k8stypes.UID) {
	if so.queue.Len() > 0 {
		fmt.Println(fmt.Sprintf("queue already populated, size: %d, not adding resource object items", so.queue.Len()))
		return
	}

	batchSize := so.updateBatchSize()
	//fmt.Println("prior to adding resource objects, batch size updated to:", batchSize)

	so.addResourcesWithBatchLimit(snapshotId, batchSize)
}

func (so *SnapshotObjects) updateBatchSize() int {
	batchLimit := so.SnapshotObjectsContext.BatchLimit
	var batchSize int

	if so.resourceObjects.Count() > batchLimit {
		batchSize = batchLimit
	} else {
		batchSize = so.resourceObjects.Count()
		if batchSize == 0 {
			batchSize = 1
		}
	}
	return batchSize
}

func (so *SnapshotObjects) addResourcesWithBatchLimit(snapshotId k8stypes.UID, batchSize int) {
	//fmt.Println("adding", so.batchSize-so.queue.Len(), "items...")
	resourceObjectItems := make([]*altc.ClusterObjectItem, 0, 0)
	for i := so.queue.Len(); i < batchSize; i++ {

		item, shutdown := so.resourceObjects.Get()
		//fmt.Println("adding resourceObject:", i+1)
		if shutdown {
			fmt.Println(fmt.Sprintf("%T shutdown", SnapshotObjects{}))
			return
		}

		resourceObjectItems = append(resourceObjectItems, item)
		so.resourceObjects.Done(item)
	}
	snapshotObject := &altc.SnapshotObject{
		ClusterName: so.SnapshotObjectsContext.ClusterName,
		SnapshotId:  snapshotId,
		Data:        resourceObjectItems,
	}
	so.queue.Add(snapshotObject)
}

func (so *SnapshotObjects) getSnapshotObject() (*altc.SnapshotObject, bool) {
	// TODO Consider making 'Populate' private and invoking the private 'populate' here
	// (why does the caller need to invoke 'Populate' and then 'Get'?)
	obj, shutdown := so.queue.Get()
	returnItem := obj.(*altc.SnapshotObject)
	return returnItem, shutdown
}

func (so *SnapshotObjects) Terminate() {
	so.queue.ShutDown()
}
