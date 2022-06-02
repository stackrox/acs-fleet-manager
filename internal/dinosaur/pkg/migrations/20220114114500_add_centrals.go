package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"gorm.io/gorm"
)

func addCentralRequest() *gormigrate.Migration {
	type CentralRequest struct {
		db.Model
		Region                        string    `json:"region"`
		ClusterID                     string    `json:"cluster_id" gorm:"index"`
		CloudProvider                 string    `json:"cloud_provider"`
		MultiAZ                       bool      `json:"multi_az"`
		Name                          string    `json:"name" gorm:"index"`
		Status                        string    `json:"status" gorm:"index"`
		SubscriptionId                string    `json:"subscription_id"`
		Owner                         string    `json:"owner" gorm:"index"`
		OwnerAccountId                string    `json:"owner_account_id"`
		Host                          string    `json:"host"`
		OrganisationId                string    `json:"organisation_id" gorm:"index"`
		FailedReason                  string    `json:"failed_reason"`
		PlacementId                   string    `json:"placement_id"`
		DesiredCentralVersion         string    `json:"desired_central_version"`
		ActualCentralVersion          string    `json:"actual_central_version"`
		DesiredCentralOperatorVersion string    `json:"desired_central_operator_version"`
		ActualCentralOperatorVersion  string    `json:"actual_central_operator_version"`
		CentralUpgrading              bool      `json:"central_upgrading"`
		CentralOperatorUpgrading      bool      `json:"central_operator_upgrading"`
		InstanceType                  string    `json:"instance_type"`
		QuotaType                     string    `json:"quota_type"`
		Routes                        api.JSON  `json:"routes"`
		RoutesCreated                 bool      `json:"routes_created"`
		Namespace                     string    `json:"namespace"`
		RoutesCreationId              string    `json:"routes_creation_id"`
		DeletionTimestamp             time.Time `json:"deletionTimestamp"`
	}

	return &gormigrate.Migration{
		ID: "20220114114500",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&CentralRequest{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&CentralRequest{})
		},
	}
}
