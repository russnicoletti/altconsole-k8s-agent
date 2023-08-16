package handlers

import (
	"altc-agent/altc"
	"altc-agent/collections"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

type handler struct {
	resourceObjects *collections.ResourceObjects
}

type Handler interface {
	cache.ResourceEventHandler
}

type HasName interface {
	Name() string
}

func NewHandler(resourceObjects *collections.ResourceObjects) Handler {
	return &handler{
		resourceObjects: resourceObjects,
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

	// Remove managed fields, if present
	metadata, ok := obj.(metav1.Object)
	if ok {
		metadata.SetManagedFields(nil)
	}

	err := h.resourceObjects.AddItem(action, resourceObject)
	if err != nil {
		fmt.Println(err.Error())
	}
}
