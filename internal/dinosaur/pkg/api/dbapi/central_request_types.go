package dbapi

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"gorm.io/gorm"
)

// CentralRequest ...
type CentralRequest struct {
	api.Meta
	Region         string `json:"region"`
	ClusterID      string `json:"cluster_id" gorm:"index"`
	CloudProvider  string `json:"cloud_provider"`
	MultiAZ        bool   `json:"multi_az"`
	Name           string `json:"name" gorm:"index"`
	Status         string `json:"status" gorm:"index"`
	SubscriptionID string `json:"subscription_id"`
	Owner          string `json:"owner" gorm:"index"` // TODO: ocm owner?
	OwnerAccountID string `json:"owner_account_id"`
	OwnerUserID    string `json:"owner_user_id"`
	// The DNS host (domain) of the Central service
	Host           string `json:"host"`
	OrganisationID string `json:"organisation_id" gorm:"index"`
	FailedReason   string `json:"failed_reason"`
	// PlacementID field should be updated every time when a CentralRequest is assigned to an OSD cluster (even if it's the same one again)
	PlacementID string `json:"placement_id"`

	DesiredCentralVersion         string `json:"desired_central_version"`
	ActualCentralVersion          string `json:"actual_central_version"`
	DesiredCentralOperatorVersion string `json:"desired_central_operator_version"`
	ActualCentralOperatorVersion  string `json:"actual_central_operator_version"`
	CentralUpgrading              bool   `json:"central_upgrading"`
	CentralOperatorUpgrading      bool   `json:"central_operator_upgrading"`
	// The type of central instance (eval or standard)
	InstanceType string `json:"instance_type"`
	// the quota service type for the central, e.g. ams, quota-management-list
	QuotaType string `json:"quota_type"`
	// Routes routes mapping for the central instance. It is an array and each item in the array contains a domain value and the corresponding route url
	Routes api.JSON `json:"routes"`
	// RoutesCreated if the routes mapping have been created in the DNS provider like Route53. Use a separate field to make it easier to query.
	RoutesCreated bool `json:"routes_created"`
	// Namespace is the namespace of the provisioned central instance.
	// We store this in the database to ensure that old centrals whose namespace contained "owner-<central-id>" information will continue to work.
	Namespace        string `json:"namespace"`
	RoutesCreationID string `json:"routes_creation_id"`
	// DeletionTimestamp stores the timestamp of the DELETE api call for the resource
	DeletionTimestamp *time.Time `json:"deletionTimestamp"`
}

// CentralList ...
type CentralList []*CentralRequest

// CentralIndex ...
type CentralIndex map[string]*CentralRequest

// Index ...
func (l CentralList) Index() CentralIndex {
	index := CentralIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
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
