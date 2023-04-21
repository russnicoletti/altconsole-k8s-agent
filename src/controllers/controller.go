package controllers

import (
	"altc-agent/altc"
	custominformers "altc-agent/informers"
	altcqueues "altc-agent/queues"
	"context"
	"fmt"
	"github.com/gogama/httpx"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/wait"
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

const (
	sendTimeout = 30 * time.Second
)

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

	c.processQueue(ctx)
}

func (c *Controller) processQueue(ctx context.Context) {

	// Give the informers time to populate their caches
	waitForInformers()

	for {
		c.clusterResourcesQ.Populate()
		clusterResources, shutdown := c.clusterResourcesQ.Get()
		if shutdown {
			fmt.Println(fmt.Sprintf("%T shutdown", altcqueues.ClusterResourcesQ{}))
			return
		}

		itemsToSend := len(clusterResources.Data)
		fmt.Println("items from clusterResources to send:", itemsToSend)
		err := c.send(ctx, clusterResources)

		// Ack the cluster resources queue item regardless of whether the item
		// was successfully sent to the server.
		//
		//  If the item was sent to the server:
		//   The item needs to be acked to indicate the queue item is finished being
		//   processed (the presence of items on the queue that are finished being
		//   processed won't prevent the queue from being shutdown).
		//
		//  If the item was not successfully sent to the server:
		//   The semantics of adding an item to a workqueue is the item won't be re-added if it
		//   is still "processing". Therefore, the item needs to be acked before being
		//   re-added.
		//
		c.clusterResourcesQ.Done(clusterResources)

		if err != nil {
			fmt.Println("ERROR: error sending resources to server:", err)
			c.clusterResourcesQ.Add(clusterResources)
			continue
		}
		fmt.Println(fmt.Sprintf("after sending %d items, resourceObjectsQ len: %d",
			itemsToSend, c.resourceObjectsQ.Len()))
		fmt.Println()
	}
}

func (c *Controller) send(ctx context.Context, clusterResources *altc.ClusterResources) error {

	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	backoff := wait.Backoff{
		Duration: 500 * time.Millisecond,
		Factor:   2,
		Jitter:   0.0,
		Steps:    4,
	}

	attempts := 0
	err := wait.ExponentialBackoffWithContext(ctx, backoff, func() (done bool, err error) {
		attempts++

		clusterResourcesJson, err := json.Marshal(*clusterResources)
		if err != nil {
			fmt.Println(fmt.Sprintf("ERROR: error marshalling clusterResources: %s", err))
			// Don't return the error from the conditionFunc, doing so will abort the retry
			return false, nil
		}
		fmt.Println(fmt.Sprintf("sending %d clusterResources items", len((*clusterResources).Data)))
		client := &httpx.Client{}
		resp, err := client.Post("http://altc-nodeserver:8080/kubernetes/resource", "application/json", clusterResourcesJson)
		if err != nil {
			fmt.Println("error hitting nodeserver endpoint:", err.Error())
			return true, nil
		}
		fmt.Println(fmt.Sprintf("response from altc-nodeserver (%d): %s", resp.StatusCode(), string(resp.Body)))
		return true, nil
	})

	return err
}

// TODO wait for informers cache to sync instead of waiting an arbitrary amount of time
func waitForInformers() {
	ticker := time.NewTicker(time.Second)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				fmt.Println(fmt.Sprintf("%s", t))
			}
		}
	}()

	const waitSeconds = 20
	fmt.Println(fmt.Sprintf("waiting %d seconds for informers cache to populate", waitSeconds))
	time.Sleep(time.Second * waitSeconds)
	ticker.Stop()
	done <- true
}
