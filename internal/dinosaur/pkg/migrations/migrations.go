package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
)

// gormigrate is a wrapper for gorm's migration functions that adds schema versioning and rollback capabilities.
// For help writing migration steps, see the gorm documentation on migrations: https://gorm.io/docs/migration.html

// Migration rules:
//
// 1. IDs are numerical timestamps that must sort ascending.
//    Use YYYYMMDDHHMM w/ 24 hour time for format
//    Example: August 21 2018 at 2:54pm would be 201808211454.
//
// 2. Include models inline with migrations to see the evolution of the object over time.
//    Using our internal type models directly in the first migration would fail in future clean installs.
//
// 3. Migrations must be backwards compatible. There are no new required fields allowed.
//    See $project_home/db/README.md
//
// 4. Create one function in a separate file that returns your Migration. Add that single function call
//    to the end of this list.
var migrations = []*gormigrate.Migration{
	addCentralRequest(),
	addClusters(),
	addLeaderLease(),
	sampleMigration(),
	addOwnerUseridToCentralRequest(),
}

func New(dbConfig *db.DatabaseConfig) (*db.Migration, func(), error) {
	return db.NewMigration(dbConfig, &gormigrate.Options{
		TableName:      "migrations",
		IDColumnName:   "id",
		IDColumnSize:   255,
		UseTransaction: false,
	}, migrations)
}
