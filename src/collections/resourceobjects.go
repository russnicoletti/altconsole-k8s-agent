package collections

import (
	"altc-agent/altc"
	"errors"
	"fmt"
	"k8s.io/client-go/util/workqueue"
)

const _resourceObjectQueueName = "altc-resourceObjectQ"

type ResourceObjects struct {
	queue workqueue.Interface
}

func NewResourceObjects() *ResourceObjects {
	queue := workqueue.NewNamed(_resourceObjectQueueName)

	return &ResourceObjects{
		queue: queue,
	}
}

func (ro *ResourceObjects) AddItem(action altc.Action, resourceObject altc.ResourceObject) error {

	clusterObjectItem, err := altc.NewClusterObjectItem(action, resourceObject)
	if err != nil {
		return errors.New(fmt.Sprintf("ERROR: unable to create %T: %s", altc.ClusterObjectItem{}, err))
	}

	ro.queue.Add(clusterObjectItem)

	return nil
}

func (ro *ResourceObjects) Terminate() {
	ro.queue.ShutDown()
}

func (ro *ResourceObjects) Count() int {
	return ro.queue.Len()
}

func (ro *ResourceObjects) Get() (*altc.ClusterObjectItem, bool) {

	obj, shutdown := ro.queue.Get()
	returnItem := obj.(*altc.ClusterObjectItem)
	return returnItem, shutdown
}

func (ro *ResourceObjects) Done(item interface{}) {
	ro.queue.Done(item)
}
