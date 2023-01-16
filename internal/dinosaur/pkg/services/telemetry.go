package services

import (
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/pkg/client/telemetry"
)

// Telemetry is the telemetry boot service.
type Telemetry struct {
	config *telemetry.TelemetryConfig
}

// NewTelemetry creates a new telemetry service instance.
func NewTelemetry(config *telemetry.TelemetryConfig) *Telemetry {
	return &Telemetry{config: config}
}

// RegisterTenant emits a group event that captures meta data of the input central instance.
func (t *Telemetry) RegisterTenant(central *dbapi.CentralRequest) {
	if t != nil && t.config.Enabled() {
		return
	}
	if telemeter := t.config.Telemeter(); telemeter != nil {
		props := map[string]any{
			"cloudAccount":   central.CloudAccountID,
			"cloudProvider":  central.CloudProvider,
			"instanceType":   central.InstanceType,
			"organisationID": central.OrganisationID,
			"tenantID":       central.ID,
		}
		telemeter.Group(central.ID, central.ID, props)
	}
}

// Start the telemetry service.
func (t *Telemetry) Start() {}

// Stop the telemetry service.
func (t *Telemetry) Stop() {
	if t == nil {
		return
	}
	if telemeter := t.config.Telemeter(); telemeter != nil {
		t.config.Telemeter().Stop()
	}
}
