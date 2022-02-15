package api

import (
	"time"

	"gorm.io/gorm"
)

type LeaderLease struct {
	Meta
	Leader    string
	LeaseType string
	Expires   *time.Time
}

type LeaderLeaseList []*LeaderLease
type LeaderLeaseIndex map[string]*LeaderLease

func (l LeaderLeaseList) Index() LeaderLeaseIndex {
	index := LeaderLeaseIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (leaderLease *LeaderLease) BeforeCreate(tx *gorm.DB) error {
	leaderLease.ID = NewID()
	return nil
}
