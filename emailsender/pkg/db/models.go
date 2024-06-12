package db

import "time"

// EmailSentByTenant represents how many emails sent by tenant
type EmailSentByTenant struct {
	TenantID  string    `gorm:"index"`
	CreatedAt time.Time `gorm:"index"` // gorm automatically set to current unix seconds on create
}
