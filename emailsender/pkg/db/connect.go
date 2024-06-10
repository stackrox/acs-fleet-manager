package db

import (
	"fmt"
	"gorm.io/gorm"
	"time"

	commonDB "github.com/stackrox/acs-fleet-manager/pkg/db"
)

// DatabaseClient defines methods for fetching or updating models in DB
type DatabaseClient interface {
	InsertEmailSentByTenant(tenantID string) error
	CountEmailSentByTenantFrom(tenantID string, from time.Time) (int64, error)
}

// DatabaseConnection contains dependency for communicating with DB
type DatabaseConnection struct {
	DB *gorm.DB
}

// NewDatabaseConnection creates a new DB connection
func NewDatabaseConnection(dbConfig *commonDB.DatabaseConfig) (*DatabaseConnection, error) {
	if dbConfig.HostFile != "" && dbConfig.PortFile != "" &&
		dbConfig.UsernameFile != "" && dbConfig.PasswordFile != "" && dbConfig.NameFile != "" {
		if err := dbConfig.ReadFiles(); err != nil {
			return nil, fmt.Errorf("failed reading db config file: %v", err)
		}
	}
	connection, _ := commonDB.NewConnectionFactory(dbConfig)
	return &DatabaseConnection{DB: connection.DB}, nil
}

// Migrate automatically migrates listed models in the database
// Documentation: https://gorm.io/docs/migration.html#Auto-Migration
func (d *DatabaseConnection) Migrate() error {
	return d.DB.AutoMigrate(&EmailSentByTenant{})
}

// InsertEmailSentByTenant returns an instance of EmailSentByTenant representing how many emails tenant sent for provided date
func (d *DatabaseConnection) InsertEmailSentByTenant(tenantID string) error {

	if result := d.DB.Create(&EmailSentByTenant{TenantID: tenantID}); result.Error != nil {
		return fmt.Errorf("failed inserting into email_sent_by_tenant table: %v", result.Error)
	}
	return nil
}

// CountEmailSentByTenantFrom counts how many emails tenant sent since provided timestamp
func (d *DatabaseConnection) CountEmailSentByTenantFrom(tenantID string, from time.Time) (int64, error) {
	var count int64
	if result := d.DB.Model(&EmailSentByTenant{}).
		Where("tenant_id = ? AND created_at > ?", tenantID, from).
		Count(&count); result.Error != nil {
		return count, fmt.Errorf("failed count items in email_sent_by_tenant: %v", result.Error)
	}
	return count, nil
}
