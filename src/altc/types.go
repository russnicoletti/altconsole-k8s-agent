package altc

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

type Action string

const (
	Add    Action = "Add"
	Update Action = "Update"
	Delete Action = "Delete"
)

type ResourceObject interface {
	runtime.Object
	metav1.Object
}

type ClusterResourceItem struct {
	Action  Action
	Kind    string
	Payload ResourceObject
}

type ClusterResources struct {
	ClusterName string                 `json:"clusterName"`
	SnapshotId  k8stypes.UID           `json:"snapshotId"`
	Data        []*ClusterResourceItem `json:"data"`
}

func NewClusterResourceItem(action Action, resourceObject ResourceObject) (*ClusterResourceItem, error) {

	kinds, _, err := scheme.Scheme.ObjectKinds(resourceObject)
	if err != nil {
		fmt.Println(fmt.Sprintf("failed to find Object %T kind: %v", resourceObject, err))
		return nil, err
	}
	if len(kinds) == 0 || kinds[0].Kind == "" {
		fmt.Println(fmt.Sprintf("ERROR: unknown Object kind for Object %T", resourceObject))
		return nil, err
	}

	return &ClusterResourceItem{
		Action:  action,
		Kind:    kinds[0].Kind,
		Payload: resourceObject,
	}, nil
}
