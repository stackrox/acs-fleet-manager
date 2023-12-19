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

func addLabelsFieldToCentralRequests() *gormigrate.Migration {
	type CentralRequest struct {
		db.Model
		Labels api.JSON `json:"labels"`
	}
	migrationID := "20231201161800"

	return &gormigrate.Migration{
		ID: migrationID,
		Migrate: func(tx *gorm.DB) error {
			return addColumnIfNotExists(tx, &CentralRequest{}, "labels")
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropColumn(&CentralRequest{}, "labels")
		},
	}
}
