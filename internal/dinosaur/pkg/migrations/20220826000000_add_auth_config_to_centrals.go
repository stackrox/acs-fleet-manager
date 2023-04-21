package migrations

import (
	"fmt"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"gorm.io/gorm"
)

func addAuthConfigToCentralRequest() *gormigrate.Migration {
	newColumns := []string{"ClientID", "ClientSecret", "Issuer"}

	type AuthConfig struct {
		ClientID     string `json:"idp_client_id"`
		ClientSecret string `json:"idp_client_secret"`
		Issuer       string `json:"idp_issuer"`
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
		ID: "20220826000000",
		Migrate: func(tx *gorm.DB) error {
			for _, col := range newColumns {
				err := tx.Migrator().AddColumn(&CentralRequest{}, col)
				if err != nil {
					return fmt.Errorf("adding new column %q: %w", col, err)
				}
			}
			return nil
		},
	}
}
