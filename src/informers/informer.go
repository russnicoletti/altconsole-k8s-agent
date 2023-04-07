package informers

import (
	"altc-agent/handlers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type Informer struct {
	informer cache.SharedInformer
	handler  handlers.Handler
}

func New(informer cache.SharedInformer, queue workqueue.Interface) *Informer {
	handler := handlers.NewHandler(queue)
	informer.AddEventHandler(handler)

	return &Informer{
		informer: informer,
		handler:  handler,
	}
}
