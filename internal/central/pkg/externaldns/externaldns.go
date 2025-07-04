package externaldns

import "github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/private"

// IsEnabled checks if the external DNS feature is enabled for the given managed central.
func IsEnabled(managedCentral private.ManagedCentral) bool {
	isEnabled, ok := managedCentral.Spec.TenantResourcesValues["externalDnsEnabled"].(bool)
	return ok && isEnabled
}
