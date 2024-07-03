package db

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	commonDB "github.com/stackrox/acs-fleet-manager/pkg/db"
)

// DatabaseClient defines methods for fetching or updating models in DB
type DatabaseClient interface {
	InsertEmailSentByTenant(tenantID string) error
	CountEmailSentByTenantSince(tenantID string, since time.Time) (int64, error)
	CleanupEmailSentByTenant(before time.Time) (int64, error)
}

// DatabaseConnection contains dependency for communicating with DB
type DatabaseConnection struct {
	DB *gorm.DB
}

// NewDatabaseConnection creates a new DB connection
func NewDatabaseConnection(dbConfig *commonDB.DatabaseConfig) *DatabaseConnection {
	connection, _ := commonDB.NewConnectionFactory(dbConfig)
	return &DatabaseConnection{DB: connection.DB}
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

// CountEmailSentByTenantSince counts how many emails tenant sent since provided timestamp
func (d *DatabaseConnection) CountEmailSentByTenantSince(tenantID string, since time.Time) (int64, error) {
	var count int64
	if result := d.DB.Model(&EmailSentByTenant{}).
		Where("tenant_id = ? AND created_at > ?", tenantID, since).
		Count(&count); result.Error != nil {
		return count, fmt.Errorf("failed count items in email_sent_by_tenant: %v", result.Error)
	}
	return count, nil
}

// CleanupEmailSentByTenant removes all EmailSendByTenant rows that were created
// before the given input time returns the number of rows affected and DB errors
func (d *DatabaseConnection) CleanupEmailSentByTenant(before time.Time) (int64, error) {
	res := d.DB.Where("created_at < ?", before).Delete(&EmailSentByTenant{})
	if err := res.Error; err != nil {
		return 0, fmt.Errorf("failed to cleanup expired emails, %w", err)
	}

	return res.RowsAffected, nil
}
