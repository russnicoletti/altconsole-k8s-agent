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
	Action  Action         `json:"action"`
	Payload ResourceObject `json:"payload"`
}

type ClusterResourceItem struct {
	Action  Action         `json:"action"`
	Kind    string         `json:"kind"`
	Payload ResourceObject `json:"payload"`
}
type ClusterResources struct {
	ClusterName string                 `json:"clusterName"`
	Data        []*ClusterResourceItem `json:"data"`
}
