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

// LeaderLeaseList ...
type LeaderLeaseList []*LeaderLease

// LeaderLeaseIndex ...
type LeaderLeaseIndex map[string]*LeaderLease

// Index ...
func (l LeaderLeaseList) Index() LeaderLeaseIndex {
	index := LeaderLeaseIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

// BeforeCreate ...
func (leaderLease *LeaderLease) BeforeCreate(tx *gorm.DB) error {
	leaderLease.ID = NewID()
	return nil
}
