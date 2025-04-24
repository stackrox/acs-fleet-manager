package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"gorm.io/gorm"
)

func addClusterAddons() *gormigrate.Migration {
	type Cluster struct {
		db.Model
		Addons api.JSON `json:"addons"`
	}

	migrationID := "20231123000000"
	colName := "Addons"

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
