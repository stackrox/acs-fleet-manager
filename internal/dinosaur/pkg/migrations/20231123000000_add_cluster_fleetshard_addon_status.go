package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"gorm.io/gorm"
)

func addClusterFleetshardAddonStatus() *gormigrate.Migration {
	type Cluster struct {
		db.Model
		FleetshardAddonStatus api.JSON `json:"fleetshard_addon_status"`
	}

	migrationID := "20231123000000"
	colName := "FleetshardAddonStatus"

	return &gormigrate.Migration{
		ID: migrationID,
		Migrate: func(tx *gorm.DB) error {
			if !tx.Migrator().HasColumn(&Cluster{}, colName) {
				if err := tx.Migrator().AddColumn(&Cluster{}, colName); err != nil {
					return errors.Wrapf(err, "adding column %s in migration %s", colName, migrationID)
				}
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if tx.Migrator().HasColumn(&Cluster{}, colName) {
				if err := tx.Migrator().DropColumn(&Cluster{}, colName); err != nil {
					return errors.Wrapf(err, "rolling back from column %s in migration %s", colName, migrationID)
				}
			}
			return nil
		},
	}
}
