package controllers

import (
	"altc-agent/altc"
	custominformers "altc-agent/informers"
	altcqueues "altc-agent/queues"
	"context"
	"fmt"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"os"
	"strconv"
	"time"
)

type Controller struct {
	custominformers         []*custominformers.Informer
	informerFactory         informers.SharedInformerFactory
	resourceObjectsQ        altcqueues.ResourceObjectsQ
	snapshotObjectsQ        altcqueues.SnapshotObjectsQ
	altcClient              *altc.Client
	batchLimit              int
	snapshotIntervalSeconds int
}

const (
	batchLimitEnv       = "BATCH_LIMIT"
	snapshotIntervalEnv = "SNAPSHOT_INTERVAL_SECONDS"
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

	batchLimit, _ := strconv.Atoi(os.Getenv(batchLimitEnv))
	snapshotIntervalSeconds, _ := strconv.Atoi(os.Getenv(snapshotIntervalEnv))

	//goland:noinspection SpellCheckingInspection
	resourceObjectsQ := altcqueues.NewResourceObjectQ()
	snapshotObjectsQ := altcqueues.NewSnapshotObjectsQ(clusterName, batchLimit, resourceObjectsQ)

	custominformers := []*custominformers.Informer{
		custominformers.New(f.Core().V1().ConfigMaps().Informer(), "ConfigMaps", resourceObjectsQ),
		custominformers.New(f.Core().V1().Endpoints().Informer(), "Endpoints", resourceObjectsQ),
		custominformers.New(f.Core().V1().Events().Informer(), "Events", resourceObjectsQ),
		custominformers.New(f.Core().V1().LimitRanges().Informer(), "LimitRanges", resourceObjectsQ),
		custominformers.New(f.Core().V1().Namespaces().Informer(), "Namespaces", resourceObjectsQ),
		custominformers.New(f.Core().V1().Nodes().Informer(), "Nodes", resourceObjectsQ),
		custominformers.New(f.Core().V1().PersistentVolumeClaims().Informer(), "PersistentVolumeClaims", resourceObjectsQ),
		custominformers.New(f.Core().V1().PodTemplates().Informer(), "PodTemplates", resourceObjectsQ),
		custominformers.New(f.Core().V1().Pods().Informer(), "Pods", resourceObjectsQ),
		custominformers.New(f.Core().V1().ReplicationControllers().Informer(), "ReplicationControllers", resourceObjectsQ),
		custominformers.New(f.Core().V1().ResourceQuotas().Informer(), "ResourceQuotas", resourceObjectsQ),
		custominformers.New(f.Core().V1().Secrets().Informer(), "Secrets", resourceObjectsQ),
		custominformers.New(f.Core().V1().ServiceAccounts().Informer(), "ServiceAccounts", resourceObjectsQ),
		custominformers.New(f.Core().V1().Services().Informer(), "Services", resourceObjectsQ),
		custominformers.New(f.Apps().V1().Deployments().Informer(), "Deployments", resourceObjectsQ),
		custominformers.New(f.Apps().V1().DaemonSets().Informer(), "DaemonSets", resourceObjectsQ),
		custominformers.New(f.Apps().V1().ReplicaSets().Informer(), "ReplicaSets", resourceObjectsQ),
		custominformers.New(f.Apps().V1().StatefulSets().Informer(), "StatefulSets", resourceObjectsQ),
		custominformers.New(f.Batch().V1().CronJobs().Informer(), "CronJobs", resourceObjectsQ),
		custominformers.New(f.Batch().V1().Jobs().Informer(), "Jobs", resourceObjectsQ),
		custominformers.New(f.Networking().V1().Ingresses().Informer(), "Ingresses", resourceObjectsQ),
		custominformers.New(f.Networking().V1().NetworkPolicies().Informer(), "NetworkPolicies", resourceObjectsQ),
		custominformers.New(f.Rbac().V1().ClusterRoles().Informer(), "ClusterRoles", resourceObjectsQ),
		custominformers.New(f.Rbac().V1().ClusterRoleBindings().Informer(), "ClusterRoleBindings", resourceObjectsQ),
		custominformers.New(f.Rbac().V1().Roles().Informer(), "Roles", resourceObjectsQ),
		custominformers.New(f.Rbac().V1().RoleBindings().Informer(), "RoleBindings", resourceObjectsQ),
		custominformers.New(f.Storage().V1().CSIStorageCapacities().Informer(), "CSIStorageCapacities", resourceObjectsQ),
	}

	return &Controller{
		custominformers:         custominformers,
		informerFactory:         f,
		resourceObjectsQ:        resourceObjectsQ,
		snapshotObjectsQ:        snapshotObjectsQ,
		altcClient:              altc.NewClient(),
		batchLimit:              batchLimit,
		snapshotIntervalSeconds: snapshotIntervalSeconds,
	}
}

func (c *Controller) Run(stopCh <-chan struct{}, ctx context.Context) {
	fmt.Println("****")
	fmt.Println("controller running")
	fmt.Println(fmt.Sprintf("cluster name: %s", c.snapshotObjectsQ.GetClusterName()))
	fmt.Println("starting informers...")
	fmt.Println("batch limit:", c.batchLimit)
	fmt.Println("snapshot interval seconds:", c.snapshotIntervalSeconds)
	fmt.Println("****")
	c.informerFactory.Start(stopCh) // runs in background

	go func() {
		<-ctx.Done()
		c.resourceObjectsQ.ShutDown()
		c.snapshotObjectsQ.ShutDown()
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
	fmt.Println("waiting for informers' caches to sync")
	err := c.waitForInformersToSync(ctx)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("informers' caches have synced")

	for {
		ready, stop := scheduleCollection(c.collectResources, time.Duration(c.snapshotIntervalSeconds)*time.Second, ctx.Done())
		select {
		case <-ready:
			//fmt.Println("cluster objects ready to be collected from resource objects queue")
			fmt.Println("Number of cluster resource objects:", c.resourceObjectsQ.Len())
			snapshotId := c.snapshotObjectsQ.Initialize()
			fmt.Println("snapshotId:", snapshotId)

			// Process all resources collected in the snapshot
			for ok := true; ok; ok = c.resourceObjectsQ.Len() != 0 {
				//fmt.Println("populating snapshotObjects queue")
				c.snapshotObjectsQ.Populate(snapshotId)
				snapshotObject, shutdown := c.snapshotObjectsQ.Get()

				if shutdown {
					fmt.Println(fmt.Sprintf("%T shutdown", altcqueues.SnapshotObjectsQ{}))
				}
				itemsToSend := len(snapshotObject.Data)
				//fmt.Println("items from snapshotObject to send:", itemsToSend)
				err := c.altcClient.Send(ctx, snapshotObject)

				// Ack the snapshotObjects queue item regardless of whether the item
				// was successfully sent to the server.
				//
				//  If the item was sent to the server:
				//   The item needs to be acked to indicate the queue item is finished being
				//   processed (the presence of items on the queue that are not finished being
				//   processed will prevent the queue from being shutdown).
				//
				//  If the item was not successfully sent to the server:
				//   The semantics of adding an item to a workqueue is such that the item won't be re-added if it
				//   is still "processing". Therefore, the item needs to be acked before being
				//   re-added.
				//
				c.snapshotObjectsQ.Done(snapshotObject)

				if err != nil {
					fmt.Println("ERROR: error sending resources to server:", err)
					c.snapshotObjectsQ.Add(snapshotObject)
					continue
				}
				fmt.Println(fmt.Sprintf("after sending %d items, resourceObjectsQ len: %d",
					itemsToSend, c.resourceObjectsQ.Len()))
			}
			break
		case <-stop:
			fmt.Println("stop collection of resources")
			return
		}
	}
}

func (c *Controller) waitForInformersToSync(ctx context.Context) error {
	cacheSyncs := make([]cache.InformerSynced, 0, len(c.custominformers))
	for _, informer := range c.custominformers {
		cacheSyncs = append(cacheSyncs, func() bool {
			hasSynced := informer.Informer.HasSynced()
			if !hasSynced {
				fmt.Println(fmt.Sprintf("Informer cache for %s has not been synced", informer.Name))
			}

			return hasSynced
		})
	}

	if !cache.WaitForCacheSync(ctx.Done(), cacheSyncs...) {
		return fmt.Errorf("informers sync failed")
	}

	return nil

}

func (c *Controller) collectResources() {
	// Shouldn't be collecting resources until all previously collected
	// resources have been sent to the server
	if c.snapshotObjectsQ.Len() != 0 {
		fmt.Println("snapshotObjectsQ is not empty, not adding additional resources")
		return
	}

	fmt.Println("collecting snapshot objects at", time.Now())
	for _, informer := range c.custominformers {
		resourcesList := informer.Informer.GetStore().List()
		fmt.Println(fmt.Sprintf("collecting %d objects from %s informer", len(resourcesList), informer.Name))
		for _, item := range resourcesList {
			informer.Handler.OnAdd(item)
		}
	}
	fmt.Println("finished collecting objects at", time.Now())
}

func scheduleCollection(collect func(), delay time.Duration, done <-chan struct{}) (<-chan bool, <-chan bool) {
	fmt.Println("scheduling snapshot object collection for", time.Now().Add(delay))

	ready := make(chan bool)
	stop := make(chan bool)
	go func() {
		for {
			select {
			case <-time.After(delay):
				collect()
				ready <- true
				return
			case <-done:
				fmt.Println("go func from scheduleCollection is stopped")
				stop <- true
				return
			}
		}
	}()
	return ready, stop
}
