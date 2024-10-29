package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"database/sql"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"gorm.io/gorm"
)

func addEnteredProvisioningAtToCentralRequest() *gormigrate.Migration {
	type CentralRequest struct {
		db.Model
		EnteredProvisioningAt sql.NullTime `json:"entered_provisioning_at"`
	}
	migrationID := "20241022160000"

	return &gormigrate.Migration{
		ID: migrationID,
		Migrate: func(tx *gorm.DB) error {
			return addColumnIfNotExists(tx, &CentralRequest{}, "entered_provisioning_at")
		},
		Rollback: func(tx *gorm.DB) error {
			return errors.Wrap(
				tx.Migrator().DropColumn(&CentralRequest{}, "entered_provisioning_at"),
				"failed to drop entered_provisioning_at column",
			)
		},
	}
}
