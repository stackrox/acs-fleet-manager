package externaldns

import "github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/private"

func IsEnabled(managedCentral private.ManagedCentral) bool {
	isEnabled, ok := managedCentral.Spec.TenantResourcesValues["externalDnsEnabled"].(bool)
	return ok && isEnabled
}
