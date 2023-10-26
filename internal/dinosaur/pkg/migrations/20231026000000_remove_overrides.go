package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"gorm.io/gorm"
)

func removeCentralAndScannerOverrides() *gormigrate.Migration {
	type CentralRequest struct {
		api.Meta
		ForceReconcile string   `json:"force_reconcile"`
		Central        api.JSON `json:"central"`
		Scanner        api.JSON `json:"scanner"`
		OperatorImage  api.JSON `json:"operator_image"`
	}

	migrationID := "20231026000000"

	return &gormigrate.Migration{
		ID: migrationID,
		Migrate: func(tx *gorm.DB) error {
			if err := dropIfColumnExists(tx, &CentralRequest{}, "force_reconcile"); err != nil {
				return err
			}
			if err := dropIfColumnExists(tx, &CentralRequest{}, "central"); err != nil {
				return err
			}
			if err := dropIfColumnExists(tx, &CentralRequest{}, "scanner"); err != nil {
				return err
			}
			return dropIfColumnExists(tx, &CentralRequest{}, "operator_image")
		},
		Rollback: func(tx *gorm.DB) error {
			return nil
		},
	}
}
