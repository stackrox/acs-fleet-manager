package services

import (
	"context"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/client/telemetry"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
)

// TenantGroupName holds the name of the Tenant group.
const TenantGroupName = "Tenant"

// TelemetryAuth is a wrapper around the user claim extraction.
//
//go:generate moq -out telemetry_moq.go . TelemetryAuth
type TelemetryAuth interface {
	getUserFromContext(ctx context.Context) (string, error)
}

// TelemetryAuthImpl is the default telemetry auth implementation.
type TelemetryAuthImpl struct{}

// NewTelemetryAuth creates a new telemetry auth.
func NewTelemetryAuth() TelemetryAuth {
	return &TelemetryAuthImpl{}
}

func (t *TelemetryAuthImpl) getUserFromContext(ctx context.Context) (string, error) {
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

// Telemetry is the telemetry boot service.
type Telemetry struct {
	auth   TelemetryAuth
	config telemetry.TelemetryConfig
}

// NewTelemetry creates a new telemetry service instance.
func NewTelemetry(auth TelemetryAuth, config telemetry.TelemetryConfig) *Telemetry {
	return &Telemetry{auth: auth, config: config}
}

func (t *Telemetry) enabled() bool {
	return t != nil && t.config != nil && t.config.Enabled()
}

// RegisterTenant emits a group event that captures meta data of the input central instance.
// Adds the token user to the tenant group.
func (t *Telemetry) RegisterTenant(ctx context.Context, central *dbapi.CentralRequest) {
	if !t.enabled() {
		return
	}

	user, err := t.auth.getUserFromContext(ctx)
	if err != nil {
		glog.Error(errors.Wrap(err, "cannot get telemetry user from context claims"))
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
	// Group call will issue a supporting Track event to force group properties
	// update.
	t.config.Telemeter().Group(props,
		telemeter.WithUserID(user),
		telemeter.WithGroups(TenantGroupName, central.ID),
	)
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

	user, err := t.auth.getUserFromContext(ctx)
	if err != nil {
		glog.Error(errors.Wrap(err, "cannot get telemetry user from context claims"))
		return
	}

	props := map[string]any{
		"Tenant ID":        tenantID,
		"Error":            errMsg,
		"Success":          requestErr == nil,
		"Is Admin Request": isAdmin,
	}
	t.config.Telemeter().Track(
		"Central Creation Requested",
		props,
		telemeter.WithUserID(user),
		telemeter.WithGroups(TenantGroupName, tenantID),
	)
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

	user, err := t.auth.getUserFromContext(ctx)
	if err != nil {
		glog.Error(errors.Wrap(err, "cannot get telemetry user from context claims"))
		return
	}

	props := map[string]any{
		"Tenant ID":        tenantID,
		"Error":            errMsg,
		"Success":          requestErr == nil,
		"Is Admin Request": isAdmin,
	}
	t.config.Telemeter().Track(
		"Central Deletion Requested",
		props,
		telemeter.WithUserID(user),
		telemeter.WithGroups(TenantGroupName, tenantID),
	)
}

// Start the telemetry service.
func (t *Telemetry) Start() {}

// Stop the telemetry service.
func (t *Telemetry) Stop() {
	if t.enabled() {
		t.config.Telemeter().Stop()
	}
}
