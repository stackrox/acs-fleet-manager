package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func sampleMigration() *gormigrate.Migration {
	// Define any types to migrate / create in the migration if desired

	return &gormigrate.Migration{
		ID: "20220114115000",
		Migrate: func(tx *gorm.DB) error {
			// Implement here the desired migration steps. Common actions that are
			// done are:
			// - Perform the creation of new data types. Usually with the
			//   tx.AutoMigrate method. See 20220114114500_add_dinosaurs.go for an
			//   example of it
			// - Perform modifications on existing data types like
			//   adding/removing/updating some of their attributes and/or values. This
			//   can be done by using AutoMigrate in some cases or by executing arbitrary SQL in
			//   others. The later can be done using the provided 'tx' object's
			//   methods. An example of this can be found in part of the
			//   20220114114503_add_leader_lease.go migration
			// - Arbitrary SQL statements. This can be done using the provided 'tx'
			//   object's methods.
			// As a good practice, all migrations should be implemented in an
			// idempotent way
			var err error
			return err
		},
		Rollback: func(tx *gorm.DB) error {
			// Implement here the logic to perform a rollback in case this migration fails.
			// It should be implement assuming that the migration could have failed in any existing
			// step
			var err error
			return err
		},
	}
}
