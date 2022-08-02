package constants

// ClusterOperation type
type ClusterOperation string

// ClusterOperationCreate ...
const (
	// ClusterOperationCreate - OpenShift/k8s cluster create operation
	ClusterOperationCreate ClusterOperation = "create"

	// ClusterOperationDelete - OpenShift/k8s cluster delete operation
	ClusterOperationDelete ClusterOperation = "delete"
)

// String ...
func (c ClusterOperation) String() string {
	return string(c)
}
