package migrations

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"gorm.io/gorm"
)

func newAMSClient(ocmConfig *ocm.OCMConfig) (ocm.Client, error) {
	err := ocmConfig.ReadFiles()
	if err != nil {
		return nil, errors.Wrap(err, "reading OCM config files")
	}
	if !ocmConfig.EnableMock {
		conn, _, err := ocm.NewOCMConnection(ocmConfig, ocmConfig.AmsURL)
		if err != nil {
			return nil, errors.Wrap(err, "creating OCM connection")
		}
		return ocm.NewClient(conn), nil
	}
	return nil, nil
}

func fetchOrgName(amsClient ocm.Client, orgID string) (string, error) {
	// Leave org name empty if client is not set.
	if amsClient == nil {
		return "", nil
	}
	org, err := amsClient.GetOrganisationFromExternalID(orgID)
	return org.Name(), err
}

func addOrganisationNameToCentralRequest(ocmConfig *ocm.OCMConfig) *gormigrate.Migration {
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
		AuthConfig
	}

	id := "202212150000"
	return &gormigrate.Migration{
		ID: id,
		Migrate: func(tx *gorm.DB) error {
			if err := tx.Migrator().AddColumn(&CentralRequest{}, "OrganisationName"); err != nil {
				return errors.Wrapf(err, "adding column OrganisationName in migration %s", id)
			}

			amsClient, err := newAMSClient(ocmConfig)
			if err != nil {
				return errors.Wrapf(err, "creating AMS client in migration %s", id)
			}

			rows, err := tx.Model(&CentralRequest{}).Where("organisation_name IS NULL").Rows()
			if err != nil {
				return errors.Wrapf(err, "fetching rows where organisation_name is NULL in migration %s", id)
			}
			defer func() {
				if err := rows.Close(); err != nil {
					panic(errors.Wrapf(err, "closing rows in migration %s", id))
				}
			}()

			for rows.Next() {
				var central CentralRequest
				if err := tx.ScanRows(rows, &central); err != nil {
					return errors.Wrapf(err, "scanning rows in migration %s", id)
				}

				orgName, err := fetchOrgName(amsClient, central.OrganisationID)
				if err != nil {
					return errors.Wrapf(err, "fetching organisation_name in migration %s", id)
				}
				if err = tx.Model(&central).Update("organisation_name", orgName).Error; err != nil {
					return errors.Wrapf(err, "updating organisation_name in migration %s", id)
				}
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Migrator().DropColumn(&CentralRequest{}, "OrganisationName"); err != nil {
				return errors.Wrapf(err, "rolling back column OrganisationName in migration %s", id)
			}
			return nil
		},
	}
}
