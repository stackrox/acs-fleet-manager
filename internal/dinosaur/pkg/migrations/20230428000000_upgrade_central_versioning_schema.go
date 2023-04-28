package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"fmt"
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"gorm.io/gorm"
)

func addCentralVersionSchema() *gormigrate.Migration {
	migrationID := "20230428000000"
	// TODO(sbaumer): Use previous_operator_version instead
	columnName := "previous_operator_version"

	return &gormigrate.Migration{
		ID: migrationID,
		Migrate: func(tx *gorm.DB) error {
			err := tx.Migrator().AddColumn(&dbapi.CentralRequest{}, columnName)
			if err != nil {
				return fmt.Errorf("failed adding column %s: %s", columnName, err)
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return nil
		},
	}
}
