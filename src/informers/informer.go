package informers

import (
	"altc-agent/handlers"
	altcqueues "altc-agent/queues"
	"k8s.io/client-go/tools/cache"
)

type Informer struct {
	Name     string
	Informer cache.SharedInformer
	Handler  handlers.Handler
}

func New(informer cache.SharedInformer, name string, resourceObjectQ altcqueues.ResourceObjectsQ) *Informer {
	handler := handlers.NewHandler(resourceObjectQ)
	// handler is invoked explicitly when processing informer cache.
	// handlers should not be invoked dynamically by informers
	/*
		informer.AddEventHandler(handler)
	*/

	return &Informer{
		Name:     name,
		Informer: informer,
		Handler:  handler,
	}
}
