package altcqueues

import (
	"altc-agent/altc"
	"errors"
	"fmt"
	"k8s.io/client-go/util/workqueue"
)

const resourceObjectQueueName = "altc-resourceObjectQ"

type ResourceObjectsQ struct {
	queue workqueue.Interface
}

func NewResourceObjectQ() ResourceObjectsQ {
	queue := workqueue.NewNamed(resourceObjectQueueName)

	return ResourceObjectsQ{
		queue: queue,
	}
}

func (q *ResourceObjectsQ) AddItem(action altc.Action, resourceObject altc.ResourceObject) error {

	clusterResourceItem, err := altc.NewClusterResourceItem(action, resourceObject)
	if err != nil {
		return errors.New(fmt.Sprintf("ERROR: unable to create %T: %s", altc.ClusterResourceItem{}, err))
	}

	q.queue.Add(clusterResourceItem)

	return nil
}

func (q *ResourceObjectsQ) ShutDown() {
	q.queue.ShutDown()
}

func (q *ResourceObjectsQ) Len() int {
	return q.queue.Len()
}

func (q *ResourceObjectsQ) Get() (item *altc.ClusterResourceItem, shutdown bool) {

	obj, shutdown := q.queue.Get()
	returnItem := obj.(*altc.ClusterResourceItem)
	return returnItem, shutdown
}

func (q *ResourceObjectsQ) Done(item interface{}) {
	q.queue.Done(item)
}
