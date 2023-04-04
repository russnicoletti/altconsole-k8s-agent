package controllers

import (
	"altc-agent/altc"
	custominformers "altc-agent/informers"
	"context"
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
	c.informerFactory.Start(stopCh) // runs in background

	fmt.Println("processing queue...")
	fmt.Println("****")

	go func() {
		<-ctx.Done()
		c.queue.ShutDown()
	}()

	c.processQueue()
}

func (c *Controller) processQueue() {
	for {
		item, shutdown := c.queue.Get()
		fmt.Println("processing queue item...")
		if shutdown {
			return
		}

		clusterResourceQueueItem, ok := item.(*altc.ClusterResourceQueueItem)
		if !ok {
			fmt.Println(fmt.Sprintf("Expected queue item to be %T, got %T", &altc.ClusterResourceQueueItem{}, item))
		}

		clusterResourceQueueItemJson, err := json.Marshal(clusterResourceQueueItem)
		if err != nil {
			fmt.Println("ERROR: error marshalling clusterResourceItem:", err)
			continue
		}

		fmt.Println(string(clusterResourceQueueItemJson))
		c.queue.Done(item)
	}
}
