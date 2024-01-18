package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"gorm.io/gorm"
)

const expirationDateLeaseType = "expiration_date_worker"

// addCentralAuthLease adds a leader lease value for the central_auth_config lease and its
// worker.
// It is similar to addLeaderLease.
func addExpirationLeaseType() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20231212140000",
		Migrate: func(tx *gorm.DB) error {
			// Set an initial already expired lease for expiration_date_worker.
			err := tx.Create(&api.LeaderLease{
				Expires:   &db.DinosaurAdditionalLeasesExpireTime,
				LeaseType: expirationDateLeaseType,
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
