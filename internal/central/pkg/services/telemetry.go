package services

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/client/telemetry"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/segment"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
)

// TenantGroupName holds the name of the Tenant group.
const TenantGroupName = "Tenant"

// segmentChancesRaiser is a sleep period for the telemeter.Group call to finish
// its 3 background attempts to set the group properties.
const segmentChancesRaiser = 6 * time.Second

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

// Telemetry is the telemetry boot service.
type Telemetry struct {
	telemeter telemeter.Telemeter
}

// NewTelemetry creates a new telemetry service instance.
func NewTelemetry(config *telemetry.TelemetryConfig) *Telemetry {
	return &Telemetry{
		telemeter: newTelemeter(*config),
	}
}

func (t *Telemetry) enabled() bool {
	return t != nil && t.telemeter != nil
}

// getTenantProperties returns the tenant group properties map.
func (t *Telemetry) getTenantProperties(central *dbapi.CentralRequest) map[string]any {
	props := map[string]any{
		"Cloud Account":   central.CloudAccountID,
		"Cloud Provider":  central.CloudProvider,
		"Instance Type":   central.InstanceType,
		"Organisation ID": central.OrganisationID,
		"Region":          central.Region,
		"Tenant ID":       central.ID,
		"Status":          central.Status,
	}
	if central.ExpiredAt.Valid {
		props["Expired At"] = central.ExpiredAt.Time.UTC().Format(time.RFC3339)
	} else {
		// An instance may loose its expiration date after quota is granted, so
		// we need to reset the group property, hence never report nil time, as
		// nil is not a value on Amplitude.
		props["Expired At"] = time.Time{}.Format(time.RFC3339)
	}
	return props
}

// trackCreationRequested emits a track event that signals the creation request of a Central instance.
func (t *Telemetry) trackCreationRequested(ctx context.Context, tenantID string, isAdmin bool, requestErr error) {
	if !t.enabled() {
		return
	}

	var errMsg string
	if requestErr != nil {
		errMsg = requestErr.Error()
	}

	user, err := getUserFromContext(ctx)
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
	t.telemeter.Track(
		"Central Creation Requested",
		props,
		telemeter.WithUserID(user),
		telemeter.WithGroup(TenantGroupName, tenantID),
	)
}

// RegisterTenant initializes the tenant group with the associated properties
// and issues a following event tracking the central creation request.
func (t *Telemetry) RegisterTenant(ctx context.Context, convCentral *dbapi.CentralRequest, isAdmin bool, err error) {
	if !t.enabled() {
		return
	}
	user, err := getUserFromContext(ctx)
	if err != nil {
		glog.Error(errors.Wrap(err, "cannot get telemetry user from context claims"))
		return
	}

	props := t.getTenantProperties(convCentral)
	// Adds the token user to the tenant group.
	// Group call will issue a supporting Track event to force group properties
	// update.
	t.telemeter.Group(
		telemeter.WithTraits(props),
		telemeter.WithUserID(user),
		telemeter.WithGroup(TenantGroupName, convCentral.ID),
	)

	go func() {
		// This is to raise the chances for the tenant group properties be
		// procesed by Segment:
		time.Sleep(segmentChancesRaiser)
		t.trackCreationRequested(ctx, convCentral.ID, isAdmin, err)
	}()
}

// UpdateTenantProperties updates tenant group properties.
func (t *Telemetry) UpdateTenantProperties(convCentral *dbapi.CentralRequest) {
	if !t.enabled() {
		return
	}
	props := t.getTenantProperties(convCentral)
	// Update tenant group properties from the name of fleet-manager backend.
	t.telemeter.Group(
		telemeter.WithTraits(props),
		telemeter.WithGroup(TenantGroupName, convCentral.ID),
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

	user, err := getUserFromContext(ctx)
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
	t.telemeter.Track(
		"Central Deletion Requested",
		props,
		telemeter.WithUserID(user),
		telemeter.WithGroup(TenantGroupName, tenantID),
	)
}

// Start the telemetry service.
func (t *Telemetry) Start() {}

// Stop the telemetry service.
func (t *Telemetry) Stop() {
	if t.enabled() {
		t.telemeter.Stop()
	}
}

// Telemeter is a wrapper interface for the telemeter interface to enable mock testing.
//
//go:generate moq -out telemetry_moq.go . Telemeter
type Telemeter interface {
	telemeter.Telemeter
}

func newTelemeter(config telemetry.TelemetryConfig) telemeter.Telemeter {
	if config.StorageKey == "" {
		return nil
	}
	return segment.NewTelemeter(
		config.StorageKey,
		config.Endpoint,
		config.ClientID,
		config.ClientName,
		config.ClientVersion,
		config.PushInterval,
		config.BatchSize,
		nil,
	)
}
