package dbapi

import (
	"encoding/json"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"gorm.io/gorm"
)

type DinosaurRequest struct {
	api.Meta
	Region         string `json:"region"`
	ClusterID      string `json:"cluster_id" gorm:"index"`
	CloudProvider  string `json:"cloud_provider"`
	MultiAZ        bool   `json:"multi_az"`
	Name           string `json:"name" gorm:"index"`
	Status         string `json:"status" gorm:"index"`
	SubscriptionId string `json:"subscription_id"`
	Owner          string `json:"owner" gorm:"index"` // TODO: ocm owner?
	OwnerAccountId string `json:"owner_account_id"`
	// The DNS host (domain) of the Dinosaur service
	Host           string `json:"host"`
	OrganisationId string `json:"organisation_id" gorm:"index"`
	FailedReason   string `json:"failed_reason"`
	// PlacementId field should be updated every time when a DinosaurRequest is assigned to an OSD cluster (even if it's the same one again)
	PlacementId string `json:"placement_id"`

	DesiredDinosaurVersion         string `json:"desired_dinosaur_version"`
	ActualDinosaurVersion          string `json:"actual_dinosaur_version"`
	DesiredDinosaurOperatorVersion string `json:"desired_dinosaur_operator_version"`
	ActualDinosaurOperatorVersion  string `json:"actual_dinosaur_operator_version"`
	DinosaurUpgrading              bool   `json:"dinosaur_upgrading"`
	DinosaurOperatorUpgrading      bool   `json:"dinosaur_operator_upgrading"`
	// The type of dinosaur instance (eval or standard)
	InstanceType string `json:"instance_type"`
	// the quota service type for the dinosaur, e.g. ams, quota-management-list
	QuotaType string `json:"quota_type"`
	// Routes routes mapping for the dinosaur instance. It is an array and each item in the array contains a domain value and the corresponding route url
	Routes api.JSON `json:"routes"`
	// RoutesCreated if the routes mapping have been created in the DNS provider like Route53. Use a separate field to make it easier to query.
	RoutesCreated bool `json:"routes_created"`
	// Namespace is the namespace of the provisioned dinosaur instance.
	// We store this in the database to ensure that old dinosaurs whose namespace contained "owner-<dinosaur-id>" information will continue to work.
	Namespace        string `json:"namespace"`
	RoutesCreationId string `json:"routes_creation_id"`
}

type DinosaurList []*DinosaurRequest
type DinosaurIndex map[string]*DinosaurRequest

func (l DinosaurList) Index() DinosaurIndex {
	index := DinosaurIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (k *DinosaurRequest) BeforeCreate(scope *gorm.DB) error {
	// To allow the id set on the DinosaurRequest object to be used. This is useful for testing purposes.
	id := k.ID
	if id == "" {
		k.ID = api.NewID()
	}
	return nil
}

func (k *DinosaurRequest) GetRoutes() ([]DataPlaneDinosaurRoute, error) {
	var routes []DataPlaneDinosaurRoute
	if k.Routes == nil {
		return routes, nil
	}
	if err := json.Unmarshal(k.Routes, &routes); err != nil {
		return nil, err
	} else {
		return routes, nil
	}
}

func (k *DinosaurRequest) SetRoutes(routes []DataPlaneDinosaurRoute) error {
	if r, err := json.Marshal(routes); err != nil {
		return err
	} else {
		k.Routes = r
		return nil
	}
}
