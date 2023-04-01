package informers

import (
	"altc-agent/handlers"
	"k8s.io/client-go/tools/cache"
)

type Informer struct {
	informer cache.SharedInformer
	handler  handlers.Handler
}

func New(informer cache.SharedInformer) *Informer {
	handler := handlers.NewHandler()
	informer.AddEventHandler(handler)

	return &Informer{
		informer: informer,
		handler:  handler,
	}
}
