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
	queue       workqueue.Interface
	batchLimit  int
	batchSize   int
	clusterName string
}

func NewClusterResourcesQ(clusterName string, batchLimit int) ClusterResourcesQ {
	queue := workqueue.NewNamed(clusterResourceQName)

	return ClusterResourcesQ{
		queue:       queue,
		batchLimit:  batchLimit,
		batchSize:   batchLimit,
		clusterName: clusterName,
	}
}

func (q *ClusterResourcesQ) AddResources(resourceObjectQ ResourceObjectQ) {
	q.UpdateBatchSize(resourceObjectQ)
	fmt.Println("prior to adding resource objects, batch size updated to:", q.batchSize)
	q.addResourcesWithBatchLimit(resourceObjectQ)
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

func (q *ClusterResourcesQ) addResourcesWithBatchLimit(resourceObjectQ ResourceObjectQ) {
	clusterResourceItems := make([]*altc.ClusterResourceItem, 0, 0)
	for i := 0; i < q.batchSize; i++ {

		item, shutdown := resourceObjectQ.Get()
		fmt.Println("adding resourceObject:", i+1)
		if shutdown {
			fmt.Println(fmt.Sprintf("%T shutdown", ClusterResourcesQ{}))
			return
		}

		clusterResourceItems = append(clusterResourceItems, item)
		// TODO need a way to re-queue clusterResourceItems when there is
		// an error sending the batch of items in the ClusterResourceQ to the server
		resourceObjectQ.Done(item)
	}
	clusterResources := &altc.ClusterResources{
		ClusterName: q.clusterName,
		Data:        clusterResourceItems,
	}
	q.queue.Add(clusterResources)
}

func (q *ClusterResourcesQ) UpdateBatchSize(resourceObjectQ ResourceObjectQ) {
	if resourceObjectQ.Len() > q.batchLimit {
		q.batchSize = q.batchLimit
	} else {
		q.batchSize = resourceObjectQ.Len()
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
