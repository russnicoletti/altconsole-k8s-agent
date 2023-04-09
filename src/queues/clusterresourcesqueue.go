package altcqueues

import (
	"altc-agent/altc"
	"fmt"
	"k8s.io/client-go/util/workqueue"
)

const clusterResourceQName = "altc-clusterResourcesQ"

// ClusterResourcesQ
// TODO consider adding ResourceObjectsQ
type ClusterResourcesQ struct {
	queue            workqueue.Interface
	resourceObjectsQ ResourceObjectsQ
	batchLimit       int
	batchSize        int
	clusterName      string
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

func (q *ClusterResourcesQ) AddResources() {
	q.UpdateBatchSize()
	fmt.Println("prior to adding resource objects, batch size updated to:", q.batchSize)
	q.addResourcesWithBatchLimit()
}

func (q *ClusterResourcesQ) ShutDown() {
	q.queue.ShutDown()
}

func (q *ClusterResourcesQ) Len() int {
	return q.queue.Len()
}

func (q *ClusterResourcesQ) Get() (resources *altc.ClusterResources, shutdown bool) {
	obj, shutdown := q.queue.Get()
	returnItem := obj.(*altc.ClusterResources)
	return returnItem, shutdown
}

func (q *ClusterResourcesQ) Done(item interface{}) {
	q.queue.Done(item)
}

func (q *ClusterResourcesQ) addResourcesWithBatchLimit() {
	clusterResourceItems := make([]*altc.ClusterResourceItem, 0, 0)
	for i := 0; i < q.batchSize; i++ {

		item, shutdown := q.resourceObjectsQ.Get()
		fmt.Println("adding resourceObject:", i+1)
		if shutdown {
			fmt.Println(fmt.Sprintf("%T shutdown", ClusterResourcesQ{}))
			return
		}

		clusterResourceItems = append(clusterResourceItems, item)
		// TODO need a way to re-queue clusterResourceItems when there is
		// an error sending the batch of items in the ClusterResourceQ to the server
		q.resourceObjectsQ.Done(item)
	}
	clusterResources := &altc.ClusterResources{
		ClusterName: q.clusterName,
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

func (q *ClusterResourcesQ) RestoreResourceObjects() {
	items, shutdown := q.Get()
	if shutdown {
		fmt.Println(fmt.Sprintf("%T shutdown", ClusterResourcesQ{}))
		return
	}
	q.resourceObjectsQ.AddItems(items.Data)
}

func (q *ClusterResourcesQ) GetBatchSize() int {
	return q.batchSize
}

func (q *ClusterResourcesQ) GetClusterName() string {
	return q.clusterName
}
