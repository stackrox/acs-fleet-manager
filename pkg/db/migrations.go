package db

import (
	"fmt"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/golang/glog"
	"gorm.io/gorm"
)

// Migration ...
type Migration struct {
	DbFactory   *ConnectionFactory
	Gormigrate  *gormigrate.Gormigrate
	GormOptions *gormigrate.Options
}

// NewMigration ...
func NewMigration(dbConfig *DatabaseConfig, gormOptions *gormigrate.Options, migrations []*gormigrate.Migration) (*Migration, func(), error) {
	err := dbConfig.ReadFiles()
	if err != nil {
		return nil, nil, err
	}
	dbFactory, cleanup := NewConnectionFactory(dbConfig)

	return &Migration{
		DbFactory:   dbFactory,
		GormOptions: gormOptions,
		Gormigrate:  gormigrate.New(dbFactory.New(), gormOptions, migrations),
	}, cleanup, nil
}

// Migrate ...
func (m *Migration) Migrate() {
	if err := m.Gormigrate.Migrate(); err != nil {
		glog.Fatalf("Could not migrate: %v", err)
	}
}

// MigrateTo Migrating to a specific migration will not seed the database, seeds are up to date with the latest
// schema based on the most recent migration
// This should be for testing purposes mainly
func (m *Migration) MigrateTo(migrationID string) {
	if err := m.Gormigrate.MigrateTo(migrationID); err != nil {
		glog.Fatalf("Could not migrate: %v", err)
	}
}

// RollbackLast ...
func (m *Migration) RollbackLast() {
	if err := m.Gormigrate.RollbackLast(); err != nil {
		glog.Fatalf("Could not migrate: %v", err)
	}
	m.deleteMigrationTableIfEmpty(m.DbFactory.New())
}

// RollbackTo ...
func (m *Migration) RollbackTo(migrationID string) {
	if err := m.Gormigrate.RollbackTo(migrationID); err != nil {
		glog.Fatalf("Could not migrate: %v", err)
	}
}

// RollbackAll Rolls back all migrations..
func (m *Migration) RollbackAll() {
	db := m.DbFactory.New()
	type Result struct {
		ID string
	}
	sql := fmt.Sprintf("SELECT %s AS id FROM %s", m.GormOptions.IDColumnName, m.GormOptions.TableName)
	for {
		var result Result
		err := db.Raw(sql).Scan(&result).Error
		if err != nil || result.ID == "" {
			break
		}
		if err := m.Gormigrate.RollbackLast(); err != nil {
			glog.Fatalf("Could not rollback last migration: %v", err)
		}
	}
	m.deleteMigrationTableIfEmpty(db)
}

func (m *Migration) deleteMigrationTableIfEmpty(db *gorm.DB) {
	if !db.Migrator().HasTable(m.GormOptions.TableName) {
		return
	}
	result := m.CountMigrationsApplied()
	if result == 0 {
		if err := db.Migrator().DropTable(m.GormOptions.TableName); err != nil {
			glog.Fatalf("Could not drop migration table: %v", err)
		}
	}
}

// CountMigrationsApplied ...
func (m *Migration) CountMigrationsApplied() int {
	db := m.DbFactory.New()
	if !db.Migrator().HasTable(m.GormOptions.TableName) {
		return 0
	}
	sql := fmt.Sprintf("SELECT count(%s) AS id FROM %s", m.GormOptions.IDColumnName, m.GormOptions.TableName)
	var count int
	if err := db.Raw(sql).Scan(&count).Error; err != nil {
		glog.Fatalf("Could not get migration count: %v", err)
	}
	return count
}

// Model represents the base model struct. All entities will have this struct embedded.
type Model struct {
	ID        string `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
