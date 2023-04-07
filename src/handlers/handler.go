package handlers

import (
	"altc-agent/altc"
	"fmt"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type handler struct {
	queue workqueue.Interface
}

type Handler interface {
	cache.ResourceEventHandler
}

type HasName interface {
	Name() string
}

func NewHandler(queue workqueue.Interface) Handler {
	return &handler{
		queue: queue,
	}
}

func (h *handler) OnAdd(obj interface{}) {
	h.handle(altc.Add, obj)
}

func (h *handler) OnUpdate(oldObj, newObj interface{}) {
	h.handle(altc.Update, newObj)
}

func (h *handler) OnDelete(obj interface{}) {
	h.handle(altc.Delete, obj)
}

func (h *handler) handle(action altc.Action, obj interface{}) {
	resourceObject, ok := obj.(altc.ResourceObject)
	if !ok {
		fmt.Println("WARN: 'obj' is not an altc.ResourceObject")
		return
	}

	clusterResourceQueueItem := &altc.ClusterResourceQueueItem{
		Action:  action,
		Payload: resourceObject,
	}

	h.queue.Add(clusterResourceQueueItem)
}
