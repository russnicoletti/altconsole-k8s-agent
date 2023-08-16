package main

import (
	"altc-agent/controllers"
	"context"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
)

func main() {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	stopCh := make(chan struct{})
	defer close(stopCh)

	clusterName := os.Getenv("CLUSTER_NAME")

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	controller := controllers.New(clientset, clusterName)
	controller.Run(stopCh, ctx)
}
