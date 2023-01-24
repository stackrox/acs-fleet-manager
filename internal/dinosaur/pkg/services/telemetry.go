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

func (t *Telemetry) enabled() bool {
	return t != nil && t.config != nil && t.config.Enabled()
}

// RegisterTenant emits a group event that captures meta data of the input central instance.
func (t *Telemetry) RegisterTenant(central *dbapi.CentralRequest) {
	if t.enabled() {
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
}

// TrackCreationRequested emits a track event that signals the creation request of a Central instance.
func (t *Telemetry) TrackCreationRequested(id string, user string, isAdmin bool, err error) {
	if t.enabled() {
		var errMsg string
		if err != nil {
			errMsg = err.Error()
		}
		props := map[string]any{
			"Tenant ID":        id,
			"Error":            errMsg,
			"Success":          err == nil,
			"Is Admin Request": isAdmin,
		}
		t.config.Telemeter().Track("Central Creation Requested", user, props)
	}
}

// TrackDeletionRequested emits a track event that signals the deletion request of a Central instance.
func (t *Telemetry) TrackDeletionRequested(id string, user string, isAdmin bool, err error) {
	if t.enabled() {
		var errMsg string
		if err != nil {
			errMsg = err.Error()
		}
		props := map[string]any{
			"Tenant ID":        id,
			"Error":            errMsg,
			"Success":          err == nil,
			"Is Admin Request": isAdmin,
		}
		t.config.Telemeter().Track("Central Deletion Requested", user, props)
	}
}

// Start the telemetry service.
func (t *Telemetry) Start() {}

// Stop the telemetry service.
func (t *Telemetry) Stop() {
	if t.enabled() {
		t.config.Telemeter().Stop()
	}
}
