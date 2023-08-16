package informers

import (
	"k8s.io/client-go/tools/cache"
)

type Informer struct {
	Name     string
	Informer cache.SharedInformer
}

func New(informer cache.SharedInformer, name string) *Informer {
	// Handler is no longer needed. It is only needed in the event-driven model, which is not being used.
	/*
		handler := handlers.NewHandler(resourceObjects)
		informer.AddEventHandler(handler)
	*/

	return &Informer{
		Name:     name,
		Informer: informer,
	}
}
