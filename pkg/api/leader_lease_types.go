package api

import (
	"time"

	"gorm.io/gorm"
)

// LeaderLease ...
type LeaderLease struct {
	Meta
	Leader    string
	LeaseType string
	Expires   *time.Time
}

// BeforeCreate ...
func (leaderLease *LeaderLease) BeforeCreate(tx *gorm.DB) error {
	leaderLease.ID = NewID()
	return nil
}
