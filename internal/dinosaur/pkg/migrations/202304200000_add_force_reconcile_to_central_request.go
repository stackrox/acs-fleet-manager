package migrations

import (
	"time"

	"github.com/golang/glog"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"gorm.io/gorm"
)

func addForceReconcileToCentralRequest() *gormigrate.Migration {
	type AuthConfig struct {
		ClientID     string `json:"idp_client_id"`
		ClientSecret string `json:"idp_client_secret"`
		Issuer       string `json:"idp_issuer"`
		ClientOrigin string `json:"client_origin"`
	}

	type CentralRequest struct {
		api.Meta
		Region           string   `json:"region"`
		ClusterID        string   `json:"cluster_id" gorm:"index"`
		CloudProvider    string   `json:"cloud_provider"`
		CloudAccountID   string   `json:"cloud_account_id"`
		MultiAZ          bool     `json:"multi_az"`
		Name             string   `json:"name" gorm:"index"`
		Status           string   `json:"status" gorm:"index"`
		SubscriptionID   string   `json:"subscription_id"`
		Owner            string   `json:"owner" gorm:"index"`
		OwnerAccountID   string   `json:"owner_account_id"`
		OwnerUserID      string   `json:"owner_user_id"`
		Host             string   `json:"host"`
		OrganisationID   string   `json:"organisation_id" gorm:"index"`
		OrganisationName string   `json:"organisation_name"`
		FailedReason     string   `json:"failed_reason"`
		PlacementID      string   `json:"placement_id"`
		Central          api.JSON `json:"central"`
		Scanner          api.JSON `json:"scanner"`

		DesiredCentralVersion         string     `json:"desired_central_version"`
		ActualCentralVersion          string     `json:"actual_central_version"`
		DesiredCentralOperatorVersion string     `json:"desired_central_operator_version"`
		ActualCentralOperatorVersion  string     `json:"actual_central_operator_version"`
		CentralUpgrading              bool       `json:"central_upgrading"`
		CentralOperatorUpgrading      bool       `json:"central_operator_upgrading"`
		InstanceType                  string     `json:"instance_type"`
		QuotaType                     string     `json:"quota_type"`
		Routes                        api.JSON   `json:"routes"`
		RoutesCreated                 bool       `json:"routes_created"`
		Namespace                     string     `json:"namespace"`
		RoutesCreationID              string     `json:"routes_creation_id"`
		DeletionTimestamp             *time.Time `json:"deletionTimestamp"`
		Internal                      bool       `json:"internal"`
		ForceReconcile                bool       `json:"force_reconcile"`
		AuthConfig
	}

	id := "202304200000"
	colName := "ForceReconcile"
	return &gormigrate.Migration{
		ID: id,
		Migrate: func(tx *gorm.DB) error {
			if !tx.Migrator().HasColumn(&CentralRequest{}, colName) {
				if err := tx.Migrator().AddColumn(&CentralRequest{}, colName); err != nil {
					return errors.Wrapf(err, "adding column %s in migration %s", colName, id)
				}
				glog.Infof("Successfully added the %s column", colName)
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if tx.Migrator().HasColumn(&CentralRequest{}, colName) {
				if err := tx.Migrator().DropColumn(&CentralRequest{}, colName); err != nil {
					return errors.Wrapf(err, "rolling back from column %s in migration %s", colName, id)
				}
				glog.Infof("Successfully removed the %s column", colName)
			}
			return nil
		},
	}
}
