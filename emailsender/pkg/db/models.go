package db

import "time"

// EmailSentByTenant represents how many emails sent by tenant
// be carefule with backwards compatibility of changes to this struct
// it is applied to the database by gorm Automigration, so changes
// to this struct merged to main will immediately affect the DB in integration
type EmailSentByTenant struct {
	TenantID  string    `gorm:"index"`
	CreatedAt time.Time `gorm:"index"` // gorm automatically set to current unix seconds on create
}
