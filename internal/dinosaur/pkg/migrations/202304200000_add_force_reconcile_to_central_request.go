package migrations

import (
	"github.com/golang/glog"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"gorm.io/gorm"
)

func addForceReconcileToCentralRequest() *gormigrate.Migration {
	type CentralRequest struct {
		api.Meta
		ForceReconcile string `json:"force_reconcile"`
	}

	id := "202304200000"
	colName := "ForceReconcile"
	return &gormigrate.Migration{
		ID: id,
		Migrate: func(tx *gorm.DB) error {
			if !tx.Migrator().HasColumn(&CentralRequest{}, colName) {
				if err := tx.Migrator().AddColumn(&CentralRequest{}, colName); err != nil {
					return errors.Wrapf(err, "adding column %s in migration %s", colName, id)
				}
				glog.Infof("Successfully added the %s column", colName)
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if tx.Migrator().HasColumn(&CentralRequest{}, colName) {
				if err := tx.Migrator().DropColumn(&CentralRequest{}, colName); err != nil {
					return errors.Wrapf(err, "rolling back from column %s in migration %s", colName, id)
				}
				glog.Infof("Successfully removed the %s column", colName)
			}
			return nil
		},
	}
}
