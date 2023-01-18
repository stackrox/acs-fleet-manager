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
	if t == nil || t.config == nil || !t.config.Enabled() {
		return
	}
	props := map[string]any{
		"Cloud Account":   central.CloudAccountID,
		"Cloud Provider":  central.CloudProvider,
		"Instance Type":   central.InstanceType,
		"Organisation ID": central.OrganisationID,
		"Region":          central.Region,
		"Tenant ID":       central.ID,
	}
	t.config.Telemeter().Group(central.ID, central.ID, props)
}

// TrackInstanceCreation emits a track event that signals the creation of a Central instance.
func (t *Telemetry) TrackInstanceCreation(central *dbapi.CentralRequest, error string) {
	if t == nil || t.config == nil || !t.config.Enabled() {
		return
	}
	props := map[string]any{
		"Error":   error,
		"Success": error == "",
	}
	t.config.Telemeter().Track("Central Creation", central.OrganisationID, props)
}

// Start the telemetry service.
func (t *Telemetry) Start() {}

// Stop the telemetry service.
func (t *Telemetry) Stop() {
	if t == nil || t.config == nil {
		return
	}
	t.config.Telemeter().Stop()
}
