package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/golang/glog"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"gorm.io/gorm"
)

func addSchedulableToClusters() *gormigrate.Migration {
	type Cluster struct {
		db.Model
		Schedulable bool `json:"schedulable"` // To be added
	}

	id := "202303231200"
	colName := "Schedulable"
	return &gormigrate.Migration{
		ID: id,
		Migrate: func(tx *gorm.DB) error {
			if !tx.Migrator().HasColumn(&Cluster{}, colName) {
				if err := tx.Migrator().AddColumn(&Cluster{}, colName); err != nil {
					return errors.Wrapf(err, "adding column %s in migration %s", colName, id)
				}
				glog.Infof("Successfully added the %s column", colName)
			}
			return nil
		},
	}
}
