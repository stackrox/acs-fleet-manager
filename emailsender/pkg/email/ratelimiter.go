package email

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/emailsender/pkg/db"
	"time"
)

const (
	windowSizeHours = 24
)

// RateLimiter defines an exact methods for rate limiter
type RateLimiter interface {
	IsAllowed(tenantID string) bool
	PersistEmailSendEvent(tenantID string) error
}

// RateLimiterService contains configuration and dependency for rate limiter
type RateLimiterService struct {
	limitPerTenant int
	dbConnection   db.DatabaseClient
}

// NewRateLimiterService creates a new instance of RateLimiterService
func NewRateLimiterService(dbConnection *db.DatabaseConnection) *RateLimiterService {
	return &RateLimiterService{
		dbConnection: dbConnection,
	}
}

// IsAllowed checks whether specified tenant can send an email for current timestamp
func (r *RateLimiterService) IsAllowed(tenantID string) bool {
	now := time.Now()
	dayAgo := now.Add(time.Duration(-windowSizeHours) * time.Hour)
	sentDuringWindow, err := r.dbConnection.CountEmailSentByTenantSince(tenantID, dayAgo)
	if err != nil {
		glog.Errorf("Cannot count sent emails during window for tenant %s: %v", tenantID, err)
		return false
	}

	if sentDuringWindow >= int64(r.limitPerTenant) {
		glog.Warningf("Reached limit for sent emails during window for tenant %s", tenantID)
		return false
	}

	return true
}

// PersistEmailSendEvent stores email sent event
func (r *RateLimiterService) PersistEmailSendEvent(tenantID string) error {
	err := r.dbConnection.InsertEmailSentByTenant(tenantID)
	if err != nil {
		return fmt.Errorf("failed register sent email: %v", err)
	}
	return nil
}
