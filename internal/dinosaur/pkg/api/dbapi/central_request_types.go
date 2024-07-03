// Package dbapi ...
package dbapi

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"gorm.io/gorm"
)

const (
	// AuthConfigStaticClientOrigin represents a RH SSO OIDC client that is the shared, static one.
	AuthConfigStaticClientOrigin = "shared_static_rhsso"
	// AuthConfigDynamicClientOrigin represents RH SSO OIDC clients that are created dynamically.
	AuthConfigDynamicClientOrigin = "dedicated_dynamic_rhsso"
)

// CentralRequest ...
type CentralRequest struct {
	api.Meta
	// Region is the cloud region the service is deployed in, i.e. us-east-1.
	Region string `json:"region"`
	// ClusterID is the data-plane cluster ID.
	ClusterID string `json:"cluster_id" gorm:"index"`
	// CloudProvider is the cloud provider the data-plane cluster is running and is used for billing customers.
	CloudProvider string `json:"cloud_provider"`
	// CloudAccountID is the billing cloud account.
	CloudAccountID string `json:"cloud_account_id"`
	// MultiAZ enables multi availability zone (AZ) support.
	MultiAZ bool `json:"multi_az"`
	// Name of the ACS instance.
	Name string `json:"name" gorm:"index"`
	// Status is the lifecycle status of the Central request. See constants.CentralRequestStatusAccepted to see
	// valid statuses.
	Status string `json:"status" gorm:"index"`
	// SubscriptionID is returned by AMS and identifies a Central instance in their system. We need it to deregister instances again from AMS.
	SubscriptionID string `json:"subscription_id"`
	// Owner is the Red Hat SSO login name of the user who created the instance. It is either the email, or the user name, depending on what the user chose to login with. It's displayed in the console UI.
	Owner string `json:"owner" gorm:"index"`
	// OwnerAccountID is used in telemetry, it is the account_id claim of the Red Hat SSO token.
	// Deprecated: Use user_id claim in telemetry.
	OwnerAccountID string `json:"owner_account_id"`
	// OwnerUserID is the subject claim (confusingly it is NOT the user_id claim) of the Red Hat SSO token.
	OwnerUserID string `json:"owner_user_id"`
	// OwnerAlternateUserID is the user_id claim of the Red Hat SSO token.
	// It is introduced to allow for a seamless migration of the subject claim format.
	OwnerAlternateUserID string `json:"owner_alternate_user_id"`

	// Instance-independent part of the Central's hostname. For example, this
	// can be `rhacs-dev.com`, `acs-stage.rhcloud.com`, etc.
	Host string `json:"host"`
	// OrganisationID identifies a customer's organisation. It is needed as an id for authn/z, and the name for observability purposes.
	OrganisationID string `json:"organisation_id" gorm:"index"`
	// OrganisationName is not unique. Its purpose is mostly human readability and observability purposes (e.g. display in dashboards).
	OrganisationName string `json:"organisation_name"`
	// FailedReason contains the reason of a Central instance failed to schedule.
	FailedReason string `json:"failed_reason"`
	// PlacementID field should be updated every time when a CentralRequest is assigned to an OSD cluster (even if it's the same one again).
	PlacementID string `json:"placement_id"`

	// The type of central instance (eval or standard).
	InstanceType string `json:"instance_type"`
	// the quota service type for the central, e.g. ams, quota-management-list.
	QuotaType string `json:"quota_type"`
	// Routes routes mapping for the central instance. It is an array and each item in the array contains a domain value and the corresponding route url.
	Routes api.JSON `json:"routes"`
	// RoutesCreated if the routes mapping have been created in the DNS provider like Route53. Use a separate field to make it easier to query.
	RoutesCreated bool `json:"routes_created"`
	// Namespace is the namespace of the provisioned central instance.
	// We store this in the database to ensure that old centrals whose namespace contained "owner-<central-id>" information will continue to work.

	// Secrets stores the encrypted secrets reported for a central tenant
	Secrets api.JSON `json:"secrets"`
	// SecretDataSum is the b64 encoded hash of plain text data of the stored secrets.
	// It used used for equality checks of secrets in the dataplane cluster with the secrets stored in DB
	SecretDataSum string `json:"secret_data_sum"`

	Namespace        string `json:"namespace"`
	RoutesCreationID string `json:"routes_creation_id"`
	// DeletionTimestamp stores the timestamp of the DELETE api call for the resource.
	DeletionTimestamp sql.NullTime `json:"deletionTimestamp"`

	// Internal will be set for instances created by internal services, such as the probe service.
	// If Internal is set to true, telemetry will be disabled for this particular instance.
	// Note: Internal cannot be set via API, but instead will be set based on the User-Agent for the central creation
	// request (see pkg/handlers/dinosaur.go).
	Internal bool `json:"internal"`

	// All we need to integrate Central with an IdP.
	AuthConfig

	// ExpiredAt contains the timestamp of when a Central instance's quota allowance was found to be expired.
	// After a grace period, the Central instance will be marked for deletion, its status will be set to 'deprovision'.
	ExpiredAt sql.NullTime `json:"expired_at"`

	// Traits is a set of random strings assigned to an instance. Some traits
	// can be hardcoded, and change some processing parameters.
	Traits pq.StringArray `json:"traits" gorm:"type:text[]"`
}

// CentralList ...
type CentralList []*CentralRequest

// AuthConfig keeps all we need to set up IdP for a Central instance.
type AuthConfig struct {
	// OIDC client ID. It is used for authenticating users in Central via connected IdP.
	ClientID string `json:"idp_client_id"`
	// OIDC client secret.
	ClientSecret string `json:"idp_client_secret"`
	// OIDC client issuer.
	Issuer string `json:"idp_issuer"`
	// Specifies whether:
	// 1) OIDC client was dynamically created via sso.redhat.com API
	// or
	// 2) We reuse static OIDC client
	ClientOrigin string `json:"client_origin"`
}

// BeforeCreate ...
func (k *CentralRequest) BeforeCreate(scope *gorm.DB) error {
	// To allow the id set on the CentralRequest object to be used. This is useful for testing purposes.
	id := k.ID
	if id == "" {
		k.ID = api.NewID()
	}
	return nil
}

// GetRoutes ...
func (k *CentralRequest) GetRoutes() ([]DataPlaneCentralRoute, error) {
	var routes []DataPlaneCentralRoute
	if k.Routes == nil {
		return routes, nil
	}
	if err := json.Unmarshal(k.Routes, &routes); err != nil {
		return nil, fmt.Errorf("unmarshalling routes from JSON: %w", err)
	}
	return routes, nil
}

// SetRoutes ...
func (k *CentralRequest) SetRoutes(routes []DataPlaneCentralRoute) error {
	r, err := json.Marshal(routes)
	if err != nil {
		return fmt.Errorf("marshalling routes into JSON: %w", err)
	}
	k.Routes = r
	return nil
}

// SetSecrets sets CentralRequest.Secret field by converting secrets to api.JSON
func (k *CentralRequest) SetSecrets(secrets map[string]string) error {
	r, err := json.Marshal(secrets)
	if err != nil {
		return fmt.Errorf("marshalling secrets into JSON: %w", err)
	}
	k.Secrets = r
	return nil
}

// GetUIHost returns host for CLI/GUI/API connections
func (k *CentralRequest) GetUIHost() string {
	if k.Host == "" {
		return ""
	}
	return fmt.Sprintf("acs-%s.%s", k.ID, k.Host)
}

// GetDataHost return host for Sensor connections
func (k *CentralRequest) GetDataHost() string {
	if k.Host == "" {
		return ""
	}
	return fmt.Sprintf("acs-data-%s.%s", k.ID, k.Host)
}

// NullTimeToTimePtr converts sql.NullTime to *time.Time.
func NullTimeToTimePtr(t sql.NullTime) *time.Time {
	if t.Valid && !t.Time.IsZero() {
		return &t.Time
	}
	return nil
}

// TimePtrToNullTime converts *time.Time to sql.NullTime.
func TimePtrToNullTime(t *time.Time) sql.NullTime {
	if t != nil && !t.IsZero() {
		return sql.NullTime{Time: *t, Valid: true}
	}
	return sql.NullTime{}
}
