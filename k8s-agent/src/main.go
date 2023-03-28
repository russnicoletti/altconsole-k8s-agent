package main

import (
	customcontrollers "altc-agent/controllers"
	"context"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
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

	ctx := signals.SetupSignalHandler()
	err = run(ctx)
	fmt.Println("agent ending with", err)
}

func run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			// Documentation
			// If Done is not yet closed, Err returns nil.
			// If Done is closed, Err returns a non-nil error explaining why:
			//  Canceled if the context was canceled
			// or
			//  DeadlineExceeded if the context's deadline passed.
			return ctx.Err()
		default:
		}
	}
}
