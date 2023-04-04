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
	ClusterName  string
	Action       Action
	ResourceType string
	Payload      ResourceObject
}

type ClusterResourceItem struct {
	ClusterName  string `json:"clusterName"`
	Action       Action `json:"action"`
	ResourceType string `json:"resourceType"`
	Payload      string `json:"payload"`
}
