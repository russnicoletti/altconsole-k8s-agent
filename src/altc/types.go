package altc

type Action string

const (
	Add    Action = "Add"
	Update Action = "Update"
	Delete Action = "Delete"
)

type ResourceType string

type ClusterResource struct {
	ClusterName  string `json:"clusterName"`
	Action       Action `"json:action"`
	ResourceType string `json:"resourceType"`
	Payload      string `json:"payload"`
}
