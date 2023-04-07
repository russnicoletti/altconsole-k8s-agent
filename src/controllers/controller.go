package controllers

import (
	"altc-agent/altc"
	custominformers "altc-agent/informers"
	altcqueues "altc-agent/queues"
	"context"
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"
	"time"
)

type Controller struct {
	custominformers  []*custominformers.Informer
	informerFactory  informers.SharedInformerFactory
	resourceQ        altcqueues.ResourceObjectQ
	clusterResources altc.ClusterResources
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
	resourceQ := altcqueues.NewResourceObjectQ()
	custominformers := []*custominformers.Informer{
		custominformers.New(f.Core().V1().Nodes().Informer(), resourceQ),
		custominformers.New(f.Core().V1().Pods().Informer(), resourceQ),
	}

	clusterResources := altc.ClusterResources{
		ClusterName: clusterName,
		Data:        []*altc.ClusterResourceItem{},
	}
	return &Controller{
		custominformers:  custominformers,
		informerFactory:  f,
		resourceQ:        resourceQ,
		clusterResources: clusterResources,
	}
}

func (c *Controller) Run(stopCh <-chan struct{}, ctx context.Context) {
	fmt.Println("****")
	fmt.Println("controller running")
	fmt.Println(fmt.Sprintf("cluster name: %s", c.clusterResources.ClusterName))
	fmt.Println("starting informers...")
	fmt.Println("****")
	c.informerFactory.Start(stopCh) // runs in background

	go func() {
		<-ctx.Done()
		c.resourceQ.ShutDown()
	}()

	c.processQueue()
}

func (c *Controller) processQueue() {

	// Give the informers time to populate their caches
	waitForInformers()

	// TODO make this configurable
	batchLimit := 5
	var batchSize = batchLimit

	if c.resourceQ.Len() < batchLimit {
		batchSize = 0
	}

	itemsToSend := 0
	fmt.Println(fmt.Sprintf("processQueue, resourceQ len: %d, batchSize: %d", c.resourceQ.Len(), batchSize))
	for {
		item, shutdown := c.resourceQ.Get()
		if shutdown {
			return
		}

		if err := c.processQueueItem(item); err != nil {
			fmt.Println(err.Error())
			continue
		}
		itemsToSend++
		fmt.Println(fmt.Sprintf("items to send: %d:", itemsToSend))
		if batchSize == 0 || itemsToSend >= batchSize {
			c.send()
			if c.resourceQ.Len() > batchLimit {
				batchSize = batchLimit
			} else {
				batchSize = c.resourceQ.Len()
			}
			fmt.Println(fmt.Sprintf("after sending %d items, resourceQ len: %d, batchSize: %d", itemsToSend, c.resourceQ.Len(), batchSize))
			fmt.Println()
			itemsToSend = 0
			continue
		}
	}
}

func AddResourceToQ(action altc.Action, resourceObject altc.ResourceObject, resourceObjQ workqueue.Interface) {

	clusterResourceItem, err := altc.NewClusterResourceItem(action, resourceObject)
	if err != nil {
		fmt.Println(fmt.Sprintf("ERROR: unable to create %T: %s", altc.ClusterResourceItem{}, err))
		return
	}

	resourceObjQ.Add(clusterResourceItem)
}

func (c *Controller) processQueueItem(item interface{}) error {
	// TODO Update the following line when the code is added to send the resource item to the server.
	// When the code is added to send the resource item to the server, the 'resourceQ.Done' call
	// should not be invoked via a defer statement because we don't want to remove the item
	// from the resourceQ when the server is unreachable (that is the point of having a resourceQ).
	// TODO Figure out how to handle resourceQ items with respect to batching the items being
	// sent to the server.
	defer c.resourceQ.Done(item)

	clusterResourceItem, ok := item.(*altc.ClusterResourceItem)
	if !ok {
		return errors.New(fmt.Sprintf("ERROR: Expected resourceQ item to be %T, got %T", &altc.ClusterResourceQueueItem{}, item))
	}
	/*
		kinds, _, err := scheme.Scheme.ObjectKinds(clusterResourceQueueItem.Payload)
		if err != nil {
			return errors.New(fmt.Sprintf("failed to find Object %T kind: %v", clusterResourceQueueItem.Payload, err))
		}
		if len(kinds) == 0 || kinds[0].Kind == "" {
			return errors.New(fmt.Sprintf("unknown Object kind for Object %T", clusterResourceQueueItem.Payload))
		}

		clusterResourceItem := &altc.ClusterResourceItem{
			Action:  clusterResourceQueueItem.Action,
			Kind:    kinds[0].Kind,
			Payload: clusterResourceQueueItem.Payload,
		}
	*/
	c.clusterResources.Data = append(c.clusterResources.Data, clusterResourceItem)
	fmt.Println(fmt.Sprintf("cluster resources data items added: %d", len(c.clusterResources.Data)))
	return nil
}

func (c *Controller) send() {
	clusterResourcesJson, err := json.Marshal(c.clusterResources)
	if err != nil {
		fmt.Println(fmt.Sprintf("ERROR: error marshalling clusterResources: %s", err))
	}
	fmt.Println()
	fmt.Println(fmt.Sprintf("sending %d clusterResources items", len(c.clusterResources.Data)))
	fmt.Println(string(clusterResourcesJson))
	fmt.Println()
	c.clusterResources = altc.ClusterResources{}
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
