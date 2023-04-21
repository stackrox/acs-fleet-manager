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

func dropSkipSchedulingFromClusters() *gormigrate.Migration {
	type Cluster struct {
		db.Model
		SkipScheduling bool `json:"skip_scheduling" gorm:"default:false"` // To be dropped
	}

	id := "202303221200"
	colName := "SkipScheduling"
	return &gormigrate.Migration{
		ID: id,
		Migrate: func(tx *gorm.DB) error {
			if tx.Migrator().HasColumn(&Cluster{}, colName) {
				if err := tx.Migrator().DropColumn(&Cluster{}, colName); err != nil {
					return errors.Wrapf(err, "rolling back from column %s in migration %s", colName, id)
				}
				glog.Infof("Successfully removed the %s column", colName)
			}
			return nil
		},
	}
}
