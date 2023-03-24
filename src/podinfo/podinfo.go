package podinfo

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
)

func CollectInfo(pods *v1.PodList) {
	fmt.Println("***************** begin pod info ***************** ")

	fmt.Println(fmt.Sprintf("There are %d pods in the cluster", len(pods.Items)))
	for i, pod := range pods.Items {
		fmt.Println(fmt.Sprintf("pod #%d", i+1))
		fmt.Println(fmt.Sprintf("name: %s", pod.Name))
		fmt.Println(fmt.Sprintf("namespace: %s", pod.Namespace))
		fmt.Println(fmt.Sprintf("restart policy: %s", pod.Spec.RestartPolicy))
		fmt.Println(fmt.Sprintf("DNS policy: %s", pod.Spec.DNSPolicy))
		fmt.Println(fmt.Sprintf("status: %s", pod.Status.Phase))
		fmt.Println(fmt.Sprintf("container image: %s", pod.Spec.Containers[0].Image))
	}
	fmt.Println("***************** end pod info ***************** ")
}
