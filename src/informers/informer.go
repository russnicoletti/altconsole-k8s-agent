package informers

import (
	"altc-agent/handlers"
	"k8s.io/client-go/tools/cache"
)

type Informer struct {
	informer cache.SharedInformer
	handler  handlers.Handler
}

func New(informer cache.SharedInformer, clusterName string) *Informer {
	handler := handlers.NewHandler(clusterName)
	informer.AddEventHandler(handler)

	return &Informer{
		informer: informer,
		handler:  handler,
	}
}
