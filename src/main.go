package main

import (
	customcontrollers "altc-agent/controllers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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

	controller := customcontrollers.New(clientset)
	controller.Run(stopCh)
}
