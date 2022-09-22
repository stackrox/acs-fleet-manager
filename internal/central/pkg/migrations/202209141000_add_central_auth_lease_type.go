package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"gorm.io/gorm"
)

const centralAuthLeaseType = "central_auth_config"

// addCentralAuthLease adds a leader lease value for the central_auth_config lease and its
// worker.
// It is similar to addLeaderLease.
func addCentralAuthLease() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202209141000",
		Migrate: func(tx *gorm.DB) error {
			// Set an initial already expired lease for central_auth_config.
			err := tx.Create(&api.LeaderLease{
				Expires:   &db.CentralAdditionalLeasesExpireTime,
				LeaseType: centralAuthLeaseType,
				Leader:    api.NewID(),
			}).Error

			if err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return nil
		},
	}
}
