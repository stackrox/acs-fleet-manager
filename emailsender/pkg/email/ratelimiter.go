package email

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/emailsender/pkg/db"
)

const (
	windowSizeHours = 24
)

// RateLimiter defines an exact methods for rate limiter
type RateLimiter interface {
	IsAllowed(tenantID string) (bool, error)
	PersistEmailSendEvent(tenantID string) error
}

// RateLimiterService contains configuration and dependency for rate limiter
type RateLimiterService struct {
	limitPerTenant int
	dbConnection   db.DatabaseClient
}

// NewRateLimiterService creates a new instance of RateLimiterService
func NewRateLimiterService(dbConnection *db.DatabaseConnection, limitPerTenant int) *RateLimiterService {
	return &RateLimiterService{
		limitPerTenant: limitPerTenant,
		dbConnection:   dbConnection,
	}
}

// IsAllowed checks whether specified tenant can send an email for current timestamp
func (r *RateLimiterService) IsAllowed(tenantID string) (bool, error) {
	now := time.Now()
	dayAgo := now.Add(time.Duration(-windowSizeHours) * time.Hour)
	sentDuringWindow, err := r.dbConnection.CountEmailSentByTenantSince(tenantID, dayAgo)
	if err != nil {
		wrappedError := fmt.Errorf("Cannot count sent emails during window for tenant %s: %v", tenantID, err)
		glog.Error(wrappedError)
		return false, wrappedError
	}

	if sentDuringWindow >= int64(r.limitPerTenant) {
		glog.Warningf("Reached limit for sent emails during window for tenant %s", tenantID)
		return false, nil
	}

	return true, nil
}

// PersistEmailSendEvent stores email sent event
func (r *RateLimiterService) PersistEmailSendEvent(tenantID string) error {
	err := r.dbConnection.InsertEmailSentByTenant(tenantID)
	if err != nil {
		return fmt.Errorf("failed register sent email: %v", err)
	}
	return nil
}
