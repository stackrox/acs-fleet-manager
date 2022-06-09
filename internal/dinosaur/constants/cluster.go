package constants

// ClusterOperation type
type ClusterOperation string

const (
	// ClusterOperationCreate - OpenShift/k8s cluster create operation
	ClusterOperationCreate ClusterOperation = "create"

	// ClusterOperationDelete - OpenShift/k8s cluster delete operation
	ClusterOperationDelete ClusterOperation = "delete"

	// The DNS prefixes used for traffic ingress
	ManagedDinosaurIngressDnsNamePrefix = "kas"
	DefaultIngressDnsNamePrefix         = "apps"
)

func (c ClusterOperation) String() string {
	return string(c)
}
