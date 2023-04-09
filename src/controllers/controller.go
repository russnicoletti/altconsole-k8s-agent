package controllers

import (
	"altc-agent/altc"
	custominformers "altc-agent/informers"
	altcqueues "altc-agent/queues"
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"time"
)

type Controller struct {
	custominformers   []*custominformers.Informer
	informerFactory   informers.SharedInformerFactory
	resourceObjectsQ  altcqueues.ResourceObjectsQ
	clusterResourcesQ altcqueues.ClusterResourcesQ
	batchLimit        int
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

	// TODO make this configurable
	batchLimit := 5

	//goland:noinspection SpellCheckingInspection
	resourceObjectsQ := altcqueues.NewResourceObjectQ()
	clusterResourceQ := altcqueues.NewClusterResourcesQ(clusterName, batchLimit, resourceObjectsQ)

	custominformers := []*custominformers.Informer{
		custominformers.New(f.Core().V1().Nodes().Informer(), resourceObjectsQ),
		custominformers.New(f.Core().V1().Pods().Informer(), resourceObjectsQ),
	}

	return &Controller{
		custominformers:   custominformers,
		informerFactory:   f,
		resourceObjectsQ:  resourceObjectsQ,
		clusterResourcesQ: clusterResourceQ,
	}
}

func (c *Controller) Run(stopCh <-chan struct{}, ctx context.Context) {
	fmt.Println("****")
	fmt.Println("controller running")
	fmt.Println(fmt.Sprintf("cluster name: %s", c.clusterResourcesQ.GetClusterName()))
	fmt.Println("starting informers...")
	fmt.Println("****")
	c.informerFactory.Start(stopCh) // runs in background

	go func() {
		<-ctx.Done()
		c.resourceObjectsQ.ShutDown()
		c.clusterResourcesQ.ShutDown()
	}()

	c.processQueue()
}

func (c *Controller) processQueue() {

	// Give the informers time to populate their caches
	waitForInformers()

	for {
		c.clusterResourcesQ.AddResources()
		clusterResources, shutdown := c.clusterResourcesQ.Get()
		if shutdown {
			fmt.Println(fmt.Sprintf("%T shutdown", altcqueues.ClusterResourcesQ{}))
			return
		}

		itemsToSend := len(clusterResources.Data)
		fmt.Println("items from clusterResources to send:", itemsToSend)
		err := c.send(clusterResources)
		if err != nil {
			fmt.Println("ERROR: error sending resources to server.", err)
			c.clusterResourcesQ.RestoreResourceObjects()
		}
		c.clusterResourcesQ.Done(clusterResources)
		fmt.Println(fmt.Sprintf("after sending %d items, resourceQ len: %d",
			itemsToSend, c.resourceObjectsQ.Len()))
		fmt.Println()
	}
}

func (c *Controller) send(clusterResources *altc.ClusterResources) error {
	clusterResourcesJson, err := json.Marshal(*clusterResources)
	if err != nil {
		fmt.Println(fmt.Sprintf("ERROR: error marshalling clusterResources: %s", err))
	}
	fmt.Println()
	fmt.Println(fmt.Sprintf("sending %d clusterResources items", len((*clusterResources).Data)))
	fmt.Println(string(clusterResourcesJson))
	fmt.Println()

	return nil
}

func waitForInformers() {
	ticker := time.NewTicker(time.Second)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				// TODO remove this (unecessary noise in logs)
				fmt.Println(fmt.Sprintf("%s", t))
			}
		}
	}()

	// TODO make this configurable
	const waitSeconds = 20
	fmt.Println(fmt.Sprintf("waiting %d seconds for informers cache to populate", waitSeconds))
	time.Sleep(time.Second * waitSeconds)
	ticker.Stop()
	done <- true
}
