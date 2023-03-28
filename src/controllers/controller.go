package controllers

import (
	custominformers "altc-agent/informers"
	"fmt"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

type Controller struct {
	custominformers []*custominformers.Informer
	factory         informers.SharedInformerFactory
}

func New(clientset *kubernetes.Clientset) *Controller {
	// Documentation
	//  The second argument is how often this informer should perform a resync.
	//  What this means is it will list all resources and rehydrate the informer's store.
	//  The reason this useful is it creates a higher guarantee that your informer's store
	//  has a perfect picture of the resources it is watching.
	//  There are situations where events can be missed entirely and resyncing every so often solves this.
	//  Setting to 0 disables resync.
	f := informers.NewSharedInformerFactory(clientset, 0)
	//goland:noinspection SpellCheckingInspection
	altcinformers := []*custominformers.Informer{
		custominformers.New(f.Core().V1().Nodes().Informer()),
		custominformers.New(f.Core().V1().Pods().Informer()),
	}

	return &Controller{
		custominformers: altcinformers,
		factory:         f,
	}
}

func (c *Controller) Run(stopCh <-chan struct{}) {
	fmt.Println("Controller, running...")
	c.factory.Start(stopCh) // runs in background
}
