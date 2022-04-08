package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addDinosaurRequest() *gormigrate.Migration {
	type DinosaurRequest struct {
		db.Model
		Region                         string   `json:"region"`
		ClusterID                      string   `json:"cluster_id" gorm:"index"`
		CloudProvider                  string   `json:"cloud_provider"`
		MultiAZ                        bool     `json:"multi_az"`
		Name                           string   `json:"name" gorm:"index"`
		Status                         string   `json:"status" gorm:"index"`
		SubscriptionId                 string   `json:"subscription_id"`
		Owner                          string   `json:"owner" gorm:"index"`
		OwnerAccountId                 string   `json:"owner_account_id"`
		Host                           string   `json:"host"`
		OrganisationId                 string   `json:"organisation_id" gorm:"index"`
		FailedReason                   string   `json:"failed_reason"`
		PlacementId                    string   `json:"placement_id"`
		DesiredDinosaurVersion         string   `json:"desired_dinosaur_version"`
		ActualDinosaurVersion          string   `json:"actual_dinosaur_version"`
		DesiredDinosaurOperatorVersion string   `json:"desired_dinosaur_operator_version"`
		ActualDinosaurOperatorVersion  string   `json:"actual_dinosaur_operator_version"`
		DinosaurUpgrading              bool     `json:"dinosaur_upgrading"`
		DinosaurOperatorUpgrading      bool     `json:"dinosaur_operator_upgrading"`
		InstanceType                   string   `json:"instance_type"`
		QuotaType                      string   `json:"quota_type"`
		Routes                         api.JSON `json:"routes"`
		RoutesCreated                  bool     `json:"routes_created"`
		Namespace                      string   `json:"namespace"`
		RoutesCreationId               string   `json:"routes_creation_id"`
	}

	return &gormigrate.Migration{
		ID: "20220114114500",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&DinosaurRequest{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&DinosaurRequest{})
		},
	}
}
