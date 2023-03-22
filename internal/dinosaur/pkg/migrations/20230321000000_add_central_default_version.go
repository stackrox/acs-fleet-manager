package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"fmt"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

const initialCentralDefaultVersion = "quay.io/rhacs-eng/stackrox-operator:3.73.1"

func addCentralDefaultVersion() *gormigrate.Migration {
	type CentralDefaultVersion struct {
		ID        uint64 `json:"id" gorm:"primarykey"`
		CreatedAt time.Time
		Version   string `json:"version"`
	}
	migrationID := "20230321000000"

	return &gormigrate.Migration{
		ID: migrationID,
		Migrate: func(tx *gorm.DB) error {
			err := tx.AutoMigrate(&CentralDefaultVersion{})
			if err != nil {
				return fmt.Errorf("migrating %s: %w", migrationID, err)
			}

			if err := tx.Create(&CentralDefaultVersion{Version: initialCentralDefaultVersion}).Error; err != nil {
				return fmt.Errorf("migrating %s: %w", migrationID, err)
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			err := tx.Migrator().DropTable(&CentralDefaultVersion{})
			if err != nil {
				return fmt.Errorf("rolling back 20230321000000: %w", err)
			}
			return nil
		},
	}
}
