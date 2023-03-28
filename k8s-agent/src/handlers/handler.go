package handlers

import (
	"fmt"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"reflect"
	//"k8s.io/apimachinery/third_party/forked/golang/bOldBytes"
	"k8s.io/client-go/tools/cache"
)

type handler struct {
}

type Handler interface {
	cache.ResourceEventHandler
}

type HasName interface {
	Name() string
}

func NewHandler() Handler {
	return &handler{}
}

func (h *handler) OnAdd(obj interface{}) {
	// Log name and type of resource
	// (The second field of all the Kubernetes objects (at least all
	// that we are interested in) is 'ObjectMeta' (which includes
	// the name of the object)
	fmt.Println(fmt.Sprintf("%s %s was created",
		reflect.ValueOf(obj).Elem().Field(1).Interface().(v1.ObjectMeta).Name,
		reflect.TypeOf(obj)))
	h.handle(obj)
}

func (h *handler) OnUpdate(oldObj, newObj interface{}) {
	fmt.Println(fmt.Sprintf("%s %s was updated",
		reflect.ValueOf(oldObj).Elem().Field(1).Interface().(v1.ObjectMeta).Name,
		reflect.TypeOf(oldObj)))
	fmt.Println("old object")
	h.handle(oldObj)

	fmt.Println("new object")
	h.handle(newObj)
}

func (h *handler) OnDelete(obj interface{}) {
	fmt.Println(fmt.Sprintf("%s %s was deleted",
		reflect.ValueOf(obj).Elem().Field(1).Interface().(v1.ObjectMeta).Name,
		reflect.TypeOf(obj)))
	h.handle(obj)
}

func (h *handler) handle(obj interface{}) {
	b, err := json.Marshal(obj)
	if err != nil {
		panic(fmt.Sprintf("error marshalling %s: %v", reflect.TypeOf(obj), err))
	}
	fmt.Println()
	fmt.Println(string(b))
	fmt.Println()
}
