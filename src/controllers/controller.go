package controllers

import (
	"altc-agent/altc"
	custominformers "altc-agent/informers"
	"context"
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"
)

type Controller struct {
	custominformers []*custominformers.Informer
	informerFactory informers.SharedInformerFactory
	clusterName     string
	queue           workqueue.Interface
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
	queue := workqueue.NewNamed("altc-queue")
	altcinformers := []*custominformers.Informer{
		custominformers.New(f.Core().V1().Nodes().Informer(), clusterName, queue),
		custominformers.New(f.Core().V1().Pods().Informer(), clusterName, queue),
	}

	return &Controller{
		custominformers: altcinformers,
		informerFactory: f,
		clusterName:     clusterName,
		queue:           queue,
	}
}

func (c *Controller) Run(stopCh <-chan struct{}, ctx context.Context) {
	fmt.Println("****")
	fmt.Println("controller running")
	fmt.Println(fmt.Sprintf("cluster name: %s", c.clusterName))
	fmt.Println("starting informers...")
	fmt.Println("****")
	c.informerFactory.Start(stopCh) // runs in background

	go func() {
		<-ctx.Done()
		c.queue.ShutDown()
	}()

	c.processQueue()
}

func (c *Controller) processQueue() {
	for {
		item, shutdown := c.queue.Get()
		if shutdown {
			return
		}

		if err := c.processQueueItem(item); err != nil {
			fmt.Println(err.Error())
			continue
		}
	}
}

func (c *Controller) processQueueItem(item interface{}) error {
	// TODO Update the following line when the code is added to send the resource item to the server.
	// When the code is added to send the resource item to the server, the 'queue.Done' call
	// should not be invoked via a defer statement because we don't want to remove the item
	// from the queue when the server is unreachable (that is the point of having a queue).
	defer c.queue.Done(item)

	fmt.Println()
	fmt.Println("processing queue item...")
	clusterResourceQueueItem, ok := item.(*altc.ClusterResourceQueueItem)
	if !ok {
		return errors.New(fmt.Sprintf("ERROR: Expected queue item to be %T, got %T", &altc.ClusterResourceQueueItem{}, item))
	}

	clusterResourceQueueItemJson, err := json.Marshal(clusterResourceQueueItem)
	if err != nil {
		return errors.New(fmt.Sprintf("ERROR: error marshalling clusterResourceItem: %s", err))
	}

	fmt.Println(string(clusterResourceQueueItemJson))

	return nil
}
