package controllers

import (
	"altc-agent/altc"
	custominformers "altc-agent/informers"
	altcqueues "altc-agent/queues"
	"context"
	"fmt"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"os"
	"strconv"
	"time"
)

type Controller struct {
	custominformers   []*custominformers.Informer
	informerFactory   informers.SharedInformerFactory
	resourceObjectsQ  altcqueues.ResourceObjectsQ
	clusterResourcesQ altcqueues.ClusterResourcesQ
	altcClient        *altc.Client
	batchLimit        int
}

const (
	batchLimitEnv = "BATCH_LIMIT"
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
	batchLimit, _ := strconv.Atoi(os.Getenv(batchLimitEnv))

	//goland:noinspection SpellCheckingInspection
	resourceObjectsQ := altcqueues.NewResourceObjectQ()
	clusterResourceQ := altcqueues.NewClusterResourcesQ(clusterName, batchLimit, resourceObjectsQ)

	custominformers := []*custominformers.Informer{
		custominformers.New(f.Core().V1().ConfigMaps().Informer(), resourceObjectsQ),
		custominformers.New(f.Core().V1().Endpoints().Informer(), resourceObjectsQ),
		custominformers.New(f.Core().V1().Events().Informer(), resourceObjectsQ),
		custominformers.New(f.Core().V1().LimitRanges().Informer(), resourceObjectsQ),
		custominformers.New(f.Core().V1().Namespaces().Informer(), resourceObjectsQ),
		custominformers.New(f.Core().V1().Nodes().Informer(), resourceObjectsQ),
		custominformers.New(f.Core().V1().PersistentVolumeClaims().Informer(), resourceObjectsQ),
		custominformers.New(f.Core().V1().PodTemplates().Informer(), resourceObjectsQ),
		custominformers.New(f.Core().V1().Pods().Informer(), resourceObjectsQ),
		custominformers.New(f.Core().V1().ReplicationControllers().Informer(), resourceObjectsQ),
		custominformers.New(f.Core().V1().ResourceQuotas().Informer(), resourceObjectsQ),
		custominformers.New(f.Core().V1().Secrets().Informer(), resourceObjectsQ),
		custominformers.New(f.Core().V1().ServiceAccounts().Informer(), resourceObjectsQ),
		custominformers.New(f.Core().V1().Services().Informer(), resourceObjectsQ),
		custominformers.New(f.Apps().V1().Deployments().Informer(), resourceObjectsQ),
		custominformers.New(f.Apps().V1().DaemonSets().Informer(), resourceObjectsQ),
		custominformers.New(f.Apps().V1().ReplicaSets().Informer(), resourceObjectsQ),
		custominformers.New(f.Apps().V1().StatefulSets().Informer(), resourceObjectsQ),
		custominformers.New(f.Batch().V1().CronJobs().Informer(), resourceObjectsQ),
		custominformers.New(f.Batch().V1().Jobs().Informer(), resourceObjectsQ),
		custominformers.New(f.Networking().V1().Ingresses().Informer(), resourceObjectsQ),
		custominformers.New(f.Networking().V1().NetworkPolicies().Informer(), resourceObjectsQ),
		custominformers.New(f.Rbac().V1().ClusterRoles().Informer(), resourceObjectsQ),
		custominformers.New(f.Rbac().V1().ClusterRoleBindings().Informer(), resourceObjectsQ),
		custominformers.New(f.Rbac().V1().Roles().Informer(), resourceObjectsQ),
		custominformers.New(f.Rbac().V1().RoleBindings().Informer(), resourceObjectsQ),
		custominformers.New(f.Storage().V1().CSIStorageCapacities().Informer(), resourceObjectsQ),
	}

	return &Controller{
		custominformers:   custominformers,
		informerFactory:   f,
		resourceObjectsQ:  resourceObjectsQ,
		clusterResourcesQ: clusterResourceQ,
		altcClient:        altc.NewClient(),
		batchLimit:        batchLimit,
	}
}

func (c *Controller) Run(stopCh <-chan struct{}, ctx context.Context) {
	fmt.Println("****")
	fmt.Println("controller running")
	fmt.Println(fmt.Sprintf("cluster name: %s", c.clusterResourcesQ.GetClusterName()))
	fmt.Println("starting informers...")
	fmt.Println("batch limit:", c.batchLimit)
	fmt.Println("****")
	c.informerFactory.Start(stopCh) // runs in background

	go func() {
		<-ctx.Done()
		c.resourceObjectsQ.ShutDown()
		c.clusterResourcesQ.ShutDown()
	}()

	err := c.altcClient.Register(ctx)
	if err == nil {
		c.processQueue(ctx)
	} else {
		fmt.Println("ERROR registering client:", err)
	}
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
		err := c.altcClient.Send(ctx, clusterResources)

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

// TODO wait for informers cache to sync instead of waiting an arbitrary amount of time
func waitForInformers() {
	ticker := time.NewTicker(time.Second)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				//fmt.Println(fmt.Sprintf("%s", t))
			}
		}
	}()

	const waitSeconds = 20

	fmt.Println()
	fmt.Println(time.Now())
	fmt.Println(fmt.Sprintf("waiting %d seconds for informers cache to populate", waitSeconds))
	time.Sleep(time.Second * waitSeconds)
	fmt.Println(time.Now())
	fmt.Println()

	ticker.Stop()
	done <- true
}
