package services

import (
	"context"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
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

func getUserFromContext(ctx context.Context) (string, error) {
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return "", errors.Wrap(err, "cannot obtain claims from context")
	}
	user, err := claims.GetUserID()
	if err != nil {
		return "", errors.Wrap(err, "cannot obtain user ID from claims")
	}
	return user, nil
}

// RegisterTenant emits a group event that captures meta data of the input central instance.
// Adds the token user to the tenant group.
func (t *Telemetry) RegisterTenant(ctx context.Context, central *dbapi.CentralRequest) {
	if !t.enabled() {
		return
	}

	user, err := getUserFromContext(ctx)
	if err != nil {
		glog.Warning(errors.Wrap(err, "cannot get telemetry user from context claims"))
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
	t.config.Telemeter().Group(central.ID, user, props)
}

// TrackCreationRequested emits a track event that signals the creation request of a Central instance.
func (t *Telemetry) TrackCreationRequested(ctx context.Context, tenantID string, isAdmin bool, requestErr error) {
	if !t.enabled() {
		return
	}

	var errMsg string
	if requestErr != nil {
		errMsg = requestErr.Error()
	}

	user, err := getUserFromContext(ctx)
	if err != nil {
		glog.Warning(errors.Wrap(err, "cannot get telemetry user from context claims"))
		return
	}

	props := map[string]any{
		"Tenant ID":        tenantID,
		"Error":            errMsg,
		"Success":          err == nil,
		"Is Admin Request": isAdmin,
	}
	t.config.Telemeter().Track("Central Creation Requested", user, props)
}

// TrackDeletionRequested emits a track event that signals the deletion request of a Central instance.
func (t *Telemetry) TrackDeletionRequested(ctx context.Context, tenantID string, isAdmin bool, requestErr error) {
	if !t.enabled() {
		return
	}

	var errMsg string
	if requestErr != nil {
		errMsg = requestErr.Error()
	}

	user, err := getUserFromContext(ctx)
	if err != nil {
		glog.Warning(errors.Wrap(err, "cannot get telemetry user from context claims"))
		return
	}

	props := map[string]any{
		"Tenant ID":        tenantID,
		"Error":            errMsg,
		"Success":          err == nil,
		"Is Admin Request": isAdmin,
	}
	t.config.Telemeter().Track("Central Deletion Requested", user, props)
}

// Start the telemetry service.
func (t *Telemetry) Start() {}

// Stop the telemetry service.
func (t *Telemetry) Stop() {
	if t.enabled() {
		t.config.Telemeter().Stop()
	}
}
