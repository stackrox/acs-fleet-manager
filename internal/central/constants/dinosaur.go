package constants

import (
	"time"
)

// DinosaurStatus type
type DinosaurStatus string

// DinosaurOperation type
type DinosaurOperation string

// DinosaurRequestStatusAccepted ...
const (
	// DinosaurRequestStatusAccepted - dinosaur request status when accepted by dinosaur worker
	DinosaurRequestStatusAccepted DinosaurStatus = "accepted"
	// DinosaurRequestStatusPreparing - dinosaur request status of a preparing dinosaur
	DinosaurRequestStatusPreparing DinosaurStatus = "preparing"
	// DinosaurRequestStatusProvisioning - dinosaur in provisioning state
	DinosaurRequestStatusProvisioning DinosaurStatus = "provisioning"
	// DinosaurRequestStatusReady - completed dinosaur request
	DinosaurRequestStatusReady DinosaurStatus = "ready"
	// DinosaurRequestStatusFailed - dinosaur request failed
	DinosaurRequestStatusFailed DinosaurStatus = "failed"
	// DinosaurRequestStatusDeprovision - dinosaur request status when to be deleted by dinosaur
	DinosaurRequestStatusDeprovision DinosaurStatus = "deprovision"
	// DinosaurRequestStatusDeleting - external resources are being deleted for the dinosaur request
	DinosaurRequestStatusDeleting DinosaurStatus = "deleting"
	// DinosaurOperationCreate - Dinosaur cluster create operations
	DinosaurOperationCreate DinosaurOperation = "create"
	// DinosaurOperationDelete = Dinosaur cluster delete operations
	DinosaurOperationDelete DinosaurOperation = "delete"
	// DinosaurOperationDeprovision = Dinosaur cluster deprovision operations
	DinosaurOperationDeprovision DinosaurOperation = "deprovision"

	// ObservabilityCanaryPodLabelKey that will be used by the observability operator to scrap metrics
	ObservabilityCanaryPodLabelKey = "managed-central-canary"

	// ObservabilityCanaryPodLabelValue the value for ObservabilityCanaryPodLabelKey
	ObservabilityCanaryPodLabelValue = "true"

	// DinosaurMaxDurationWithProvisioningErrs the maximum duration a Dinosaur request
	// might be in provisioning state while receiving 5XX errors
	DinosaurMaxDurationWithProvisioningErrs = 5 * time.Minute

	// AcceptedDinosaurMaxRetryDuration the maximum duration, in minutes, where Fleet Manager
	// will retry reconciliation of a Dinosaur request in an 'accepted' state
	AcceptedDinosaurMaxRetryDuration = 5 * time.Minute
)

// ordinals - Used to decide if a status comes after or before a given state
var ordinals = map[string]int{
	DinosaurRequestStatusAccepted.String():     0,
	DinosaurRequestStatusPreparing.String():    10,
	DinosaurRequestStatusProvisioning.String(): 20,
	DinosaurRequestStatusReady.String():        30,
	DinosaurRequestStatusDeprovision.String():  40,
	DinosaurRequestStatusDeleting.String():     50,
	DinosaurRequestStatusFailed.String():       500,
}

// NamespaceLabels contains labels that indicates if a namespace is a managed application services namespace.
// A namespace with these labels will be scrapped by the Observability operator to retrieve metrics
var NamespaceLabels = map[string]string{
	"mas-managed": "true",
}

// String ...
func (k DinosaurOperation) String() string {
	return string(k)
}

// String DinosaurStatus Methods
func (k DinosaurStatus) String() string {
	return string(k)
}

// CompareTo - Compare this status with the given status returning an int. The result will be 0 if k==k1, -1 if k < k1, and +1 if k > k1
func (k DinosaurStatus) CompareTo(k1 DinosaurStatus) int {
	ordinalK := ordinals[k.String()]
	ordinalK1 := ordinals[k1.String()]

	switch {
	case ordinalK == ordinalK1:
		return 0
	case ordinalK > ordinalK1:
		return 1
	default:
		return -1
	}
}

// GetUpdateableStatuses ...
func GetUpdateableStatuses() []string {
	return []string{
		DinosaurRequestStatusPreparing.String(),
		DinosaurRequestStatusProvisioning.String(),
		DinosaurRequestStatusFailed.String(),
		DinosaurRequestStatusReady.String(),
	}
}
