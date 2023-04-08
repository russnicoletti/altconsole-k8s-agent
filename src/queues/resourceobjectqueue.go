package altcqueues

import (
	"altc-agent/altc"
	"errors"
	"fmt"
	"k8s.io/client-go/util/workqueue"
)

const resourceObjectQueueName = "altc-resourceObjectQ"

type ResourceObjectQ struct {
	queue workqueue.Interface
}

func NewResourceObjectQ() ResourceObjectQ {
	queue := workqueue.NewNamed(resourceObjectQueueName)

	return ResourceObjectQ{
		queue: queue,
	}
}

func (q *ResourceObjectQ) AddItem(action altc.Action, resourceObject altc.ResourceObject) error {

	clusterResourceItem, err := altc.NewClusterResourceItem(action, resourceObject)
	if err != nil {
		return errors.New(fmt.Sprintf("ERROR: unable to create %T: %s", altc.ClusterResourceItem{}, err))
	}

	q.queue.Add(clusterResourceItem)

	return nil
}

func (q *ResourceObjectQ) ShutDown() {
	q.queue.ShutDown()
}

func (q *ResourceObjectQ) Len() int {
	return q.queue.Len()
}

func (q *ResourceObjectQ) Get() (item *altc.ClusterResourceItem, shutdown bool) {

	obj, shutdown := q.queue.Get()
	returnItem := obj.(*altc.ClusterResourceItem)
	return returnItem, shutdown
}

func (q *ResourceObjectQ) Done(item interface{}) {
	q.queue.Done(item)
}
