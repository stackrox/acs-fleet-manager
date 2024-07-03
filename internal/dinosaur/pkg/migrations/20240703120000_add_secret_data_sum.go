package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"gorm.io/gorm"
)

func addSecretDataSum() *gormigrate.Migration {
	type CentralRequest struct {
		db.Model
		SecretDataSum string `json:"secret_data_sum"`
	}
	migrationID := "20240703120000"

	return &gormigrate.Migration{
		ID: migrationID,
		Migrate: func(tx *gorm.DB) error {
			return addColumnIfNotExists(tx, &CentralRequest{}, "secret_data_sum")
		},
		Rollback: func(tx *gorm.DB) error {
			return errors.Wrap(
				tx.Migrator().DropColumn(&CentralRequest{}, "secret_data_sum"),
				"failed to drop secret_data_sum column",
			)
		},
	}
}
