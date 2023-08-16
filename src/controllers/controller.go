package controllers

import (
	"altc-agent/altc"
	"altc-agent/collections"
	altcinformers "altc-agent/informers"
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"os"
	"strconv"
	"time"
)

type Controller struct {
	informers       []*altcinformers.Informer
	informerFactory informers.SharedInformerFactory
	resourceObjects *collections.ResourceObjects
	snapshotObjects *collections.SnapshotObjects
	client          *altc.Client
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
	f := informers.NewSharedInformerFactory(clientset, time.Duration(30*time.Minute))

	informersList := []*altcinformers.Informer{
		altcinformers.New(f.Core().V1().ConfigMaps().Informer(), "ConfigMaps"),
		altcinformers.New(f.Core().V1().Endpoints().Informer(), "Endpoints"),
		altcinformers.New(f.Core().V1().Events().Informer(), "Events"),
		altcinformers.New(f.Core().V1().LimitRanges().Informer(), "LimitRanges"),
		altcinformers.New(f.Core().V1().Namespaces().Informer(), "Namespaces"),
		altcinformers.New(f.Core().V1().Nodes().Informer(), "Nodes"),
		altcinformers.New(f.Core().V1().PersistentVolumeClaims().Informer(), "PersistentVolumeClaims"),
		altcinformers.New(f.Core().V1().PodTemplates().Informer(), "PodTemplates"),
		altcinformers.New(f.Core().V1().Pods().Informer(), "Pods"),
		altcinformers.New(f.Core().V1().ReplicationControllers().Informer(), "ReplicationControllers"),
		altcinformers.New(f.Core().V1().ResourceQuotas().Informer(), "ResourceQuotas"),
		altcinformers.New(f.Core().V1().Secrets().Informer(), "Secrets"),
		altcinformers.New(f.Core().V1().ServiceAccounts().Informer(), "ServiceAccounts"),
		altcinformers.New(f.Core().V1().Services().Informer(), "Services"),
		altcinformers.New(f.Apps().V1().Deployments().Informer(), "Deployments"),
		altcinformers.New(f.Apps().V1().DaemonSets().Informer(), "DaemonSets"),
		altcinformers.New(f.Apps().V1().ReplicaSets().Informer(), "ReplicaSets"),
		altcinformers.New(f.Apps().V1().StatefulSets().Informer(), "StatefulSets"),
		altcinformers.New(f.Batch().V1().CronJobs().Informer(), "CronJobs"),
		altcinformers.New(f.Batch().V1().Jobs().Informer(), "Jobs"),
		altcinformers.New(f.Networking().V1().Ingresses().Informer(), "Ingresses"),
		altcinformers.New(f.Networking().V1().NetworkPolicies().Informer(), "NetworkPolicies"),
		altcinformers.New(f.Rbac().V1().ClusterRoles().Informer(), "ClusterRoles"),
		altcinformers.New(f.Rbac().V1().ClusterRoleBindings().Informer(), "ClusterRoleBindings"),
		altcinformers.New(f.Rbac().V1().Roles().Informer(), "Roles"),
		altcinformers.New(f.Rbac().V1().RoleBindings().Informer(), "RoleBindings"),
		altcinformers.New(f.Storage().V1().CSIStorageCapacities().Informer(), "CSIStorageCapacities"),
	}

	batchLimit, _ := strconv.Atoi(os.Getenv(batchLimitEnv))
	snapshotIntervalSeconds, _ := strconv.Atoi(os.Getenv(snapshotIntervalEnv))

	context := collections.SnapshotObjectsContext{
		BatchLimit:              batchLimit,
		SnapshotIntervalSeconds: snapshotIntervalSeconds,
		ClusterName:             clusterName,
	}

	resourceObjects := collections.NewResourceObjects()
	snapshotObjects := collections.NewSnapshotObjects(resourceObjects, informersList, context)

	return &Controller{
		informers:       informersList,
		informerFactory: f,
		resourceObjects: resourceObjects,
		snapshotObjects: snapshotObjects,
		client:          altc.NewClient(),
	}
}

func (c *Controller) Run(stopCh <-chan struct{}, ctx context.Context) {
	fmt.Println("****")
	fmt.Println("controller running")
	snapshotObjectsContextBytes, _ := json.Marshal(c.snapshotObjects.SnapshotObjectsContext)
	fmt.Println("snapshot objects context:", string(snapshotObjectsContextBytes))
	fmt.Println("starting informers...")
	fmt.Println("****")
	c.informerFactory.Start(stopCh) // runs in background

	go func() {
		<-ctx.Done()
		c.resourceObjects.Terminate()
		c.snapshotObjects.Terminate()
	}()

	if err := c.client.Register(ctx); err != nil {
		fmt.Println("ERROR registering client:", err)
		return
	}
	c.run(ctx)
}

func (c *Controller) run(ctx context.Context) {

	// Give the informers time to populate their caches
	fmt.Println("waiting for informers' caches to sync")
	if err := c.waitForInformersToSync(ctx); err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println("informers' caches have synced")

	c.snapshotObjects.Loop(ctx)
}

func (c *Controller) waitForInformersToSync(ctx context.Context) error {
	cacheSyncs := make([]cache.InformerSynced, 0, len(c.informers))
	for _, informer := range c.informers {
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
