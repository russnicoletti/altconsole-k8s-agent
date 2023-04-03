package controllers

import (
	custominformers "altc-agent/informers"
	"fmt"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

type Controller struct {
	custominformers []*custominformers.Informer
	informerFactory informers.SharedInformerFactory
	clusterName     string
}

func New(clientset *kubernetes.Clientset, clusterName string) *Controller {
	// Documentation
	//  The second argument is how often this informer should perform a resync.
	//  What this means is it will list all resources and rehydrate the informer's store.
	//  The reason this is useful is it creates a higher guarantee that your informer's store
	//  has a perfect picture of the resources it is watching.
	//  There are situations where events can be missed entirely and resyncing every so often solves this.
	//  Setting to 0 disables resync.
	f := informers.NewSharedInformerFactory(clientset, 0)
	//goland:noinspection SpellCheckingInspection
	altcinformers := []*custominformers.Informer{
		custominformers.New(f.Core().V1().Nodes().Informer(), clusterName),
		custominformers.New(f.Core().V1().Pods().Informer(), clusterName),
	}

	return &Controller{
		custominformers: altcinformers,
		informerFactory: f,
		clusterName:     clusterName,
	}
}

func (c *Controller) Run(stopCh <-chan struct{}) {
	fmt.Println("****")
	fmt.Println("controller running")
	fmt.Println(fmt.Sprintf("cluster name: %s", c.clusterName))
	fmt.Println("starting informers...")
	fmt.Println("****")
	c.informerFactory.Start(stopCh) // runs in background
}
