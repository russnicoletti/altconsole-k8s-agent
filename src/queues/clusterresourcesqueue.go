package altcqueues

import (
	"altc-agent/altc"
	"fmt"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/util/workqueue"
)

const clusterResourceQName = "altc-clusterResourcesQ"

type ClusterResourcesQ struct {
	queue            workqueue.Interface
	resourceObjectsQ ResourceObjectsQ
	batchLimit       int
	batchSize        int
	clusterName      string
	clusterResources *altc.ClusterResources
}

func NewClusterResourcesQ(clusterName string, batchLimit int, resourceObjectsQ ResourceObjectsQ) ClusterResourcesQ {
	queue := workqueue.NewNamed(clusterResourceQName)

	return ClusterResourcesQ{
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
// for all the clusterResources created from this queue -- until a subsequent `Initialize` call is made.
func (q *ClusterResourcesQ) Initialize() k8stypes.UID {
	return uuid.NewUUID()
}

// Populate
//
// Add items to the cluster resources queue, respecting
// the batch size. If the queue is already populated,
// does not add any additional resources.
func (q *ClusterResourcesQ) Populate(snapshotId k8stypes.UID) {
	if q.queue.Len() > 0 {
		fmt.Println(fmt.Sprintf("queue already populated, size: %d, not adding resource object items", q.queue.Len()))
		return
	}

	q.UpdateBatchSize()
	//fmt.Println("prior to adding resource objects, batch size updated to:", q.batchSize)
	q.addResourcesWithBatchLimit(snapshotId)
}

func (q *ClusterResourcesQ) ShutDown() {
	q.queue.ShutDown()
}

func (q *ClusterResourcesQ) Len() int {
	return q.queue.Len()
}

func (q *ClusterResourcesQ) Add(item *altc.ClusterResources) {
	q.queue.Add(item)
}

func (q *ClusterResourcesQ) Get() (resources *altc.ClusterResources, shutdown bool) {
	// TODO Consider making 'Populate' private and invoking the private 'populate' here
	// (why does the caller need to invoke 'Populate' and then 'Get'?)
	obj, shutdown := q.queue.Get()
	returnItem := obj.(*altc.ClusterResources)
	return returnItem, shutdown
}

func (q *ClusterResourcesQ) Done(item interface{}) {
	q.queue.Done(item)
}

func (q *ClusterResourcesQ) addResourcesWithBatchLimit(snapshotId k8stypes.UID) {
	//fmt.Println("adding", q.batchSize-q.queue.Len(), "items...")
	clusterResourceItems := make([]*altc.ClusterResourceItem, 0, 0)
	for i := q.queue.Len(); i < q.batchSize; i++ {

		item, shutdown := q.resourceObjectsQ.Get()
		//fmt.Println("adding resourceObject:", i+1)
		if shutdown {
			fmt.Println(fmt.Sprintf("%T shutdown", ClusterResourcesQ{}))
			return
		}

		clusterResourceItems = append(clusterResourceItems, item)
		q.resourceObjectsQ.Done(item)
	}
	clusterResources := &altc.ClusterResources{
		ClusterName: q.clusterName,
		SnapshotId:  snapshotId,
		Data:        clusterResourceItems,
	}
	q.queue.Add(clusterResources)
}

func (q *ClusterResourcesQ) UpdateBatchSize() {
	if q.resourceObjectsQ.Len() > q.batchLimit {
		q.batchSize = q.batchLimit
	} else {
		q.batchSize = q.resourceObjectsQ.Len()
		if q.batchSize == 0 {
			q.batchSize = 1
		}
	}
}

func (q *ClusterResourcesQ) GetBatchSize() int {
	return q.batchSize
}

func (q *ClusterResourcesQ) GetClusterName() string {
	return q.clusterName
}
