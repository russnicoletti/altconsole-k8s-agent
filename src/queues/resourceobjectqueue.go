package altcqueues

import (
	"altc-agent/altc"
	"errors"
	"fmt"
	"k8s.io/client-go/util/workqueue"
)

const queueName = "altc-resourceQ"

type ResourceObjectQ struct {
	queue workqueue.Interface
}

func NewResourceObjectQ() ResourceObjectQ {
	queue := workqueue.NewNamed(queueName)

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

func (q *ResourceObjectQ) Get() (item interface{}, shutdown bool) {
	return q.queue.Get()
}

func (q *ResourceObjectQ) Done(item interface{}) {
	q.queue.Done(item)
}
