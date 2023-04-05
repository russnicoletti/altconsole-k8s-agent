package handlers

import (
	"altc-agent/altc"
	"fmt"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type handler struct {
	clusterName string
	queue       workqueue.Interface
}

type Handler interface {
	cache.ResourceEventHandler
}

type HasName interface {
	Name() string
}

func NewHandler(clusterName string, queue workqueue.Interface) Handler {
	return &handler{
		clusterName: clusterName,
		queue:       queue,
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
		fmt.Println("'obj' is not an altc.ResourceObject")
	}
	kinds, _, err := scheme.Scheme.ObjectKinds(resourceObject)
	if err != nil {
		fmt.Println(fmt.Sprintf("failed to find Object %T kind: %v", resourceObject, err))
	}
	if len(kinds) == 0 || kinds[0].Kind == "" {
		fmt.Println(fmt.Sprintf("unknown Object kind for Object %T", resourceObject))
		return
	}

	clusterResourceQueueItem := &altc.ClusterResourceQueueItem{
		ClusterName: h.clusterName,
		Action:      action,
		Kind:        kinds[0].Kind,
		Payload:     resourceObject,
	}

	h.queue.Add(clusterResourceQueueItem)
}
