package migrations

import (
	"fmt"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"gorm.io/gorm"
)

func addClientOriginToCentralRequest() *gormigrate.Migration {
	type AuthConfig struct {
		ClientID     string `json:"idp_client_id"`
		ClientSecret string `json:"idp_client_secret"`
		Issuer       string `json:"idp_issuer"`
		ClientOrigin string `json:"client_origin" gorm:"default:shared_static_rhsso"` // Currently only static clients
		// have been used to provision centrals, hence we set it to shared_static_rhsso by default when no value previously existed.
	}

	type CentralRequest struct {
		db.Model
		Region                        string
		ClusterID                     string
		CloudProvider                 string
		MultiAZ                       bool
		Name                          string
		Status                        string
		SubscriptionID                string
		Owner                         string
		OwnerAccountID                string
		OwnerUserID                   string
		Host                          string
		OrganisationID                string
		FailedReason                  string
		PlacementID                   string
		Central                       api.JSON
		Scanner                       api.JSON
		DesiredCentralVersion         string
		ActualCentralVersion          string
		DesiredCentralOperatorVersion string
		ActualCentralOperatorVersion  string
		CentralUpgrading              bool
		CentralOperatorUpgrading      bool
		InstanceType                  string
		QuotaType                     string
		Routes                        api.JSON
		RoutesCreated                 bool
		Namespace                     string
		RoutesCreationID              string
		DeletionTimestamp             *time.Time
		AuthConfig
	}

	return &gormigrate.Migration{
		ID: "202205101700",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&CentralRequest{}); err != nil {
				return fmt.Errorf("adding new colum ClientOrigin in migration 202205101700: %w", err)
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Migrator().DropColumn(&CentralRequest{}, "ClientOrigin"); err != nil {
				return fmt.Errorf("rolling back new column ClientOrigin in migration 202205101700: %w", err)
			}
			return nil
		},
	}
}
