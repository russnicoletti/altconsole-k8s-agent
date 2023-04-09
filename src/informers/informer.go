package informers

import (
	"altc-agent/handlers"
	altcqueues "altc-agent/queues"
	"k8s.io/client-go/tools/cache"
)

type Informer struct {
	informer cache.SharedInformer
	handler  handlers.Handler
}

func New(informer cache.SharedInformer, resourceObjectQ altcqueues.ResourceObjectsQ) *Informer {
	handler := handlers.NewHandler(resourceObjectQ)
	informer.AddEventHandler(handler)

	return &Informer{
		informer: informer,
		handler:  handler,
	}
}
