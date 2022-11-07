// Package constants ...
package constants

import (
	"time"
)

// CentralStatus type
type CentralStatus string

// CentralOperation type
type CentralOperation string

// CentralRequestStatusAccepted ...
const (
	// CentralRequestStatusAccepted - central request status when accepted by central worker
	CentralRequestStatusAccepted CentralStatus = "accepted"
	// CentralRequestStatusPreparing - central request status of a preparing central
	CentralRequestStatusPreparing CentralStatus = "preparing"
	// CentralRequestStatusProvisioning - central in provisioning state
	CentralRequestStatusProvisioning CentralStatus = "provisioning"
	// CentralRequestStatusReady - completed central request
	CentralRequestStatusReady CentralStatus = "ready"
	// CentralRequestStatusFailed - central request failed
	CentralRequestStatusFailed CentralStatus = "failed"
	// CentralRequestStatusDeprovision - central request status when to be deleted by central
	CentralRequestStatusDeprovision CentralStatus = "deprovision"
	// CentralRequestStatusDeleting - external resources are being deleted for the central request
	CentralRequestStatusDeleting CentralStatus = "deleting"
	// CentralOperationCreate - Central cluster create operations
	CentralOperationCreate CentralOperation = "create"
	// CentralOperationDelete = Central cluster delete operations
	CentralOperationDelete CentralOperation = "delete"
	// CentralOperationDeprovision = Central cluster deprovision operations
	CentralOperationDeprovision CentralOperation = "deprovision"

	// ObservabilityCanaryPodLabelKey that will be used by the observability operator to scrap metrics
	ObservabilityCanaryPodLabelKey = "managed-central-canary"

	// ObservabilityCanaryPodLabelValue the value for ObservabilityCanaryPodLabelKey
	ObservabilityCanaryPodLabelValue = "true"

	// CentralMaxDurationWithProvisioningErrs the maximum duration a Central request
	// might be in provisioning state while receiving 5XX errors
	CentralMaxDurationWithProvisioningErrs = 5 * time.Minute

	// AcceptedCentralMaxRetryDuration the maximum duration, in minutes, where Fleet Manager
	// will retry reconciliation of a Central request in an 'accepted' state
	AcceptedCentralMaxRetryDuration = 5 * time.Minute
)

// ordinals - Used to decide if a status comes after or before a given state
var ordinals = map[string]int{
	CentralRequestStatusAccepted.String():     0,
	CentralRequestStatusPreparing.String():    10,
	CentralRequestStatusProvisioning.String(): 20,
	CentralRequestStatusReady.String():        30,
	CentralRequestStatusDeprovision.String():  40,
	CentralRequestStatusDeleting.String():     50,
	CentralRequestStatusFailed.String():       500,
}

// NamespaceLabels contains labels that indicates if a namespace is a managed application services namespace.
// A namespace with these labels will be scrapped by the Observability operator to retrieve metrics
var NamespaceLabels = map[string]string{
	"mas-managed": "true",
}

// String ...
func (k CentralOperation) String() string {
	return string(k)
}

// String CentralStatus Methods
func (k CentralStatus) String() string {
	return string(k)
}

// CompareTo - Compare this status with the given status returning an int. The result will be 0 if k==k1, -1 if k < k1, and +1 if k > k1
func (k CentralStatus) CompareTo(k1 CentralStatus) int {
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
		CentralRequestStatusPreparing.String(),
		CentralRequestStatusProvisioning.String(),
		CentralRequestStatusFailed.String(),
		CentralRequestStatusReady.String(),
	}
}
