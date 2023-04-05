package altc

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Action string

const (
	Add    Action = "Add"
	Update Action = "Update"
	Delete Action = "Delete"
)

type ResourceType string

type ResourceObject interface {
	runtime.Object
	metav1.Object
}

type ClusterResourceQueueItem struct {
	ClusterName string         `json:"clusterName"`
	Action      Action         `json:"action"`
	Kind        string         `json:"kind"`
	Payload     ResourceObject `json:"payload"`
}
