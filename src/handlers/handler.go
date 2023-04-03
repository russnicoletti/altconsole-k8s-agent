package handlers

import (
	"altc-agent/altc"
	"fmt"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/tools/cache"
	"reflect"
)

type handler struct {
	clusterName string
}

type Handler interface {
	cache.ResourceEventHandler
}

type HasName interface {
	Name() string
}

func NewHandler(clusterName string) Handler {
	return &handler{
		clusterName: clusterName,
	}
}

func (h *handler) OnAdd(obj interface{}) {
	h.handle(altc.Add, obj)
}

func (h *handler) OnUpdate(oldObj, newObj interface{}) {
	fmt.Println("old object")
	h.handle(altc.Update, oldObj)

	fmt.Println("new object")
	h.handle(altc.Update, newObj)
}

func (h *handler) OnDelete(obj interface{}) {
	h.handle(altc.Delete, obj)
}

func (h *handler) handle(action altc.Action, obj interface{}) {
	resourceJson, err := json.Marshal(obj)
	if err != nil {
		panic(fmt.Sprintf("error marshalling %s: %v", reflect.TypeOf(obj), err))
	}
	clusterResource := altc.ClusterResource{
		ClusterName:  h.clusterName,
		Action:       action,
		ResourceType: reflect.TypeOf(obj).Elem().Name(),
		Payload:      string(resourceJson),
	}

	clusterResourceJson, err := json.Marshal(clusterResource)
	if err != nil {
		panic(fmt.Sprintf("error marshalling clusterResourceObj: %v", err))
	}
	fmt.Println()
	fmt.Println(string(clusterResourceJson))
	fmt.Println()
}
