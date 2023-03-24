/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Note: the example only works with the code within the same release/branch.
package main

import (
	customcontrollers "altc-agent/controllers"
	"context"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
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
	fmt.Println("creating controller...")
	controller := customcontrollers.New(clientset)
	fmt.Println("starting controller...")
	controller.Run(stopCh)

	/*
		podInformer := factory.Core().V1().Pods()
		sharedIndexPodInformer := podInformer.Informer()
	*/

	/*
		var i = 0
		informerSyncer := func() bool {
			hasSynced := sharedIndexPodInformer.HasSynced()
			if !hasSynced {
				fmt.Println(fmt.Sprintf("pod informer has not synced (%d)", i))
				i += 1
			}

			return hasSynced
		}

		// sync
		fmt.Println("waiting for informer cache to sync...")
		if !cache.WaitForCacheSync(stopCh, informerSyncer) {
			runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		}

		fmt.Println("Collecting snapshot from pod informer...")
		for _, item := range sharedIndexPodInformer.GetStore().List() {
			pod, _ := item.(*v1.Pod)
			fmt.Println("Pod: ", pod.Name, "namespace:", pod.Namespace)
		}
		fmt.Println("Finished collecting snapshot from pod informer...")

		sharedIndexPodInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod, _ := obj.(*v1.Pod)
				fmt.Println("pod ", pod.Name, "was created -", pod.Namespace)
			},
			UpdateFunc: func(oldObj interface{}, newObj interface{}) {
				newPodObj, _ := newObj.(*v1.Pod)
				fmt.Println("pod ", newPodObj.Name, "was updated", time.Now())
				bNewBytes, err := json.Marshal(newPodObj)
				if err != nil {
					panic(fmt.Sprintf("error marshalling %s: %v", newPodObj.Name, err))
				}
				fmt.Println("new pod object:")
				fmt.Println(string(bNewBytes))

				oldPodObj, _ := oldObj.(*v1.Pod)
				bOldBytes, _ := json.Marshal(oldPodObj)
				fmt.Println("old pod object:")
				fmt.Println(string(bOldBytes))
			},
			DeleteFunc: func(obj interface{}) {
				pod, _ := obj.(*v1.Pod)
				fmt.Println("pod ", pod.Name, "was deleted -", pod.Namespace)
			},
		})
	*/
	ctx := signals.SetupSignalHandler()
	err = run(ctx)
	fmt.Println("program ending with", err)
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
