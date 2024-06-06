package db

import (
	"fmt"
	"gorm.io/gorm"
	"time"

	commonDB "github.com/stackrox/acs-fleet-manager/pkg/db"
)

// DatabaseClient defines methods for fetching or updating models in DB
type DatabaseClient interface {
	GetEmailSentByTenant(tenantID string, date time.Time) (*EmailSentByTenant, error)
	UpdateEmailSentByTenant(tenantID string, date time.Time, amount int) error
	GetEmailSentPerSecond() (*EmailSentPerSecond, error)
	UpdateEmailSentPerSecond(amount int) error
}

// DatabaseConnection contains dependency for communicating with DB
type DatabaseConnection struct {
	DB *gorm.DB
}

// NewDatabaseConnection creates a new DB connection
func NewDatabaseConnection(host string, port int, user, password, database, SSLMode string) (*DatabaseConnection, error) {
	dbConfig := &commonDB.DatabaseConfig{
		Host:     host,
		Port:     port,
		Username: user,
		Password: password, // pragma: allowlist secret
		Name:     database,
		SSLMode:  SSLMode,
	}
	connection, _ := commonDB.NewConnectionFactory(dbConfig)
	return &DatabaseConnection{DB: connection.DB}, nil
}

// Migrate automatically migrates listed models in the database
// Documentation: https://gorm.io/docs/migration.html#Auto-Migration
func (d *DatabaseConnection) Migrate() error {
	return d.DB.AutoMigrate(&EmailSentPerSecond{}, &EmailSentByTenant{})
}

// GetEmailSentByTenant returns an instance of EmailSentByTenant representing how many emails tenant sent for provided date
// Note: date uses only days
func (d *DatabaseConnection) GetEmailSentByTenant(tenantID string, date time.Time) (*EmailSentByTenant, error) {
	var emailSentByTenant EmailSentByTenant
	d.DB.FirstOrCreate(&emailSentByTenant, &EmailSentByTenant{TenantID: tenantID, Date: onlyDate(date)})
	return &emailSentByTenant, nil
}

// UpdateEmailSentByTenant updates how many emails sent by tenant for provided date
// Note: date uses only days
func (d *DatabaseConnection) UpdateEmailSentByTenant(tenantID string, date time.Time, amount int) error {
	if result := d.DB.Model(&EmailSentByTenant{}).
		Where("tenant_id = ? and date = ?", tenantID, onlyDate(date)).
		Update("amount", amount); result.Error != nil {
		return fmt.Errorf("failed updating email_sent_by_tenant table: %v", result.Error)
	}
	return nil
}

// GetEmailSentPerSecond returns how many emails sent for the last second
func (d *DatabaseConnection) GetEmailSentPerSecond() (*EmailSentPerSecond, error) {
	var emailSentPerSecond EmailSentPerSecond
	d.DB.FirstOrCreate(&emailSentPerSecond, &EmailSentPerSecond{})
	return &emailSentPerSecond, nil
}

// UpdateEmailSentPerSecond updates how many emails sent for the last second
func (d *DatabaseConnection) UpdateEmailSentPerSecond(amount int) error {
	if result := d.DB.Save(&EmailSentPerSecond{ID: 1, Amount: amount}); result.Error != nil {
		return fmt.Errorf("failed updating email_sent_per_second table: %v", result.Error)
	}
	return nil
}

func onlyDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
