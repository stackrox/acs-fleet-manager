package migrations

import (
	"fmt"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"gorm.io/gorm"
)

// This migration is needed to switch from mistakenly added "shared_static_sso" client origin
// to "shared_static_rhsso".
func changeCentralClientOrigin() *gormigrate.Migration {
	type AuthConfig struct {
		ClientID     string `json:"idp_client_id"`
		ClientSecret string `json:"idp_client_secret"`
		Issuer       string `json:"idp_issuer"`
		ClientOrigin string `json:"client_origin"`
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
		ID: "202210271100",
		Migrate: func(tx *gorm.DB) error {
			err := tx.Model(&CentralRequest{}).
				Where("client_origin = ?", "shared_static_sso").
				Update("client_origin", "shared_static_rhsso").Error

			if err != nil {
				return fmt.Errorf("setting shared_static_rhsso instead of shared_static_sso as value for "+
					"ClientOrigin in migration 202210271100: %w", err)
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return nil
		},
	}
}
