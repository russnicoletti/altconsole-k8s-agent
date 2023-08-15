package altcqueues

import (
	"altc-agent/altc"
	"fmt"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/util/workqueue"
)

const snapshotObjectsQName = "altc-snapshotObjectsQ"

type SnapshotObjectsQ struct {
	queue            workqueue.Interface
	resourceObjectsQ ResourceObjectsQ
	batchLimit       int
	batchSize        int
	clusterName      string
	snapshotObjects  *altc.SnapshotObject
}

func NewSnapshotObjectsQ(clusterName string, batchLimit int, resourceObjectsQ ResourceObjectsQ) SnapshotObjectsQ {
	queue := workqueue.NewNamed(snapshotObjectsQName)

	return SnapshotObjectsQ{
		queue:            queue,
		resourceObjectsQ: resourceObjectsQ,
		batchLimit:       batchLimit,
		batchSize:        batchLimit,
		clusterName:      clusterName,
	}
}

// Initialize
//
// This function should be called once for each snapshot. The function creates a snapshotId, which will be used
// for all the snapshotObjects created from this queue -- until a subsequent `Initialize` call is made.
func (q *SnapshotObjectsQ) Initialize() k8stypes.UID {
	return uuid.NewUUID()
}

// Populate
//
// Add items to the snapshot objects queue, respecting
// the batch size. If the queue is already populated,
// does not add any additional resources.
func (q *SnapshotObjectsQ) Populate(snapshotId k8stypes.UID) {
	if q.queue.Len() > 0 {
		fmt.Println(fmt.Sprintf("queue already populated, size: %d, not adding resource object items", q.queue.Len()))
		return
	}

	q.UpdateBatchSize()
	//fmt.Println("prior to adding resource objects, batch size updated to:", q.batchSize)
	q.addResourcesWithBatchLimit(snapshotId)
}

func (q *SnapshotObjectsQ) ShutDown() {
	q.queue.ShutDown()
}

func (q *SnapshotObjectsQ) Len() int {
	return q.queue.Len()
}

func (q *SnapshotObjectsQ) Add(item *altc.SnapshotObject) {
	q.queue.Add(item)
}

func (q *SnapshotObjectsQ) Get() (resources *altc.SnapshotObject, shutdown bool) {
	// TODO Consider making 'Populate' private and invoking the private 'populate' here
	// (why does the caller need to invoke 'Populate' and then 'Get'?)
	obj, shutdown := q.queue.Get()
	returnItem := obj.(*altc.SnapshotObject)
	return returnItem, shutdown
}

func (q *SnapshotObjectsQ) Done(item interface{}) {
	q.queue.Done(item)
}

func (q *SnapshotObjectsQ) addResourcesWithBatchLimit(snapshotId k8stypes.UID) {
	//fmt.Println("adding", q.batchSize-q.queue.Len(), "items...")
	clusterObjectItems := make([]*altc.ClusterObjectItem, 0, 0)
	for i := q.queue.Len(); i < q.batchSize; i++ {

		item, shutdown := q.resourceObjectsQ.Get()
		//fmt.Println("adding resourceObject:", i+1)
		if shutdown {
			fmt.Println(fmt.Sprintf("%T shutdown", SnapshotObjectsQ{}))
			return
		}

		clusterObjectItems = append(clusterObjectItems, item)
		q.resourceObjectsQ.Done(item)
	}
	snapshotObject := &altc.SnapshotObject{
		ClusterName: q.clusterName,
		SnapshotId:  snapshotId,
		Data:        clusterObjectItems,
	}
	q.queue.Add(snapshotObject)
}

func (q *SnapshotObjectsQ) UpdateBatchSize() {
	if q.resourceObjectsQ.Len() > q.batchLimit {
		q.batchSize = q.batchLimit
	} else {
		q.batchSize = q.resourceObjectsQ.Len()
		if q.batchSize == 0 {
			q.batchSize = 1
		}
	}
}

func (q *SnapshotObjectsQ) GetBatchSize() int {
	return q.batchSize
}

func (q *SnapshotObjectsQ) GetClusterName() string {
	return q.clusterName
}
