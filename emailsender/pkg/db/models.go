package db

import "time"

// EmailSentPerSecond represent how many emails sent for the last second
type EmailSentPerSecond struct {
	ID        uint // primary key
	UpdatedAt int  // gorm automatically  set to current unix seconds on update. It is equal to zero b default
	Amount    int
}

// EmailSentByTenant represents how many emails sent by tenant
type EmailSentByTenant struct {
	TenantID string    `gorm:"index"`
	Date     time.Time `gorm:"index"`
	Amount   int
}
