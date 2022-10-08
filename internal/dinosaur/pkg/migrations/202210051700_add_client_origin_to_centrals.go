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
		ID: "202210051700",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.Migrator().AddColumn(&CentralRequest{}, "ClientOrigin"); err != nil {
				return fmt.Errorf("adding new colum ClientOrigin in migration 202210051700: %w", err)
			}

			// Update only the central requests which do not already specify client_origin after migrating the table
			// schema. For central requests which do not specify it, we can assume shared_static_sso, since the field
			// was not available before adding the possibility to create centrals with a dynamic OIDC client.
			// **Note**: This update is _not_ done within a transaction since we pass `UseTransaction=false` as
			// gormigrate option, meaning this will be on a best-effort basis, but the risk of not doing it within a
			// transaction is minor.
			err := tx.Model(&CentralRequest{}).
				Where("client_origin IS NULL").
				Update("client_origin", "shared_static_sso").Error

			if err != nil {
				return fmt.Errorf("setting shared_static_sso as initial value for new column "+
					"ClientOrigin in migration 202210051700: %w", err)
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Migrator().DropColumn(&CentralRequest{}, "ClientOrigin"); err != nil {
				return fmt.Errorf("rolling back new column ClientOrigin in migration 202210051700: %w", err)
			}
			return nil
		},
	}
}
