package shared

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestParametersSHA256Sum(t *testing.T) {
	RegisterTestingT(t)
	addon := Addon{
		Parameters: map[string]string{
			"acscsEnvironment":                      "test",
			"fleetshardSyncFleetManagerEndpoint":    "http://localhost:8000",
			"fleetshardSyncResourcesLimitsCpu":      "500m",
			"fleetshardSyncResourcesLimitsMemory":   "512Mi",
			"fleetshardSyncResourcesRequestsCpu":    "200m",
			"fleetshardSyncResourcesRequestsMemory": "512Mi",
		},
	}

	Expect(addon.Parameters.SHA256Sum()).To(Equal("d6b805642f9227dd0424814728a96894b301d2d97d0a197e5fe8735ea5ea15ed")) // pragma: allowlist secret
}
