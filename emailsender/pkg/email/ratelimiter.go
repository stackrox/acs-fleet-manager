package email

import (
	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/emailsender/pkg/db"
	"time"
)

// RateLimiter defines an exact methods for rate limiter
type RateLimiter interface {
	Allow(tenantID string) bool
}

// RateLimiterService contains configuration and dependency for rate limiter
type RateLimiterService struct {
	limitPerSecond       int
	limitPerDayPerTenant int
	dbConnection         db.DatabaseClient
}

// NewRateLimiterService creates a new instance of RateLimiterService
func NewRateLimiterService(dbConnection *db.DatabaseConnection) *RateLimiterService {
	return &RateLimiterService{
		dbConnection: dbConnection,
	}
}

// Allow calculates whether an email may send now for specific tenant for current timestamp
func (r *RateLimiterService) Allow(tenantID string) bool {
	now := time.Now()
	nowSeconds := now.Unix()
	allowedPerSecond := r.allowRatePerSecond(tenantID, int(nowSeconds))
	if !allowedPerSecond {
		return false
	}
	allowedPerTenant := r.allowRatePerTenant(tenantID, now)

	return allowedPerTenant
}

func (r *RateLimiterService) allowRatePerSecond(tenantID string, now int) bool {
	emailPerSecond, err := r.dbConnection.GetEmailSentPerSecond()
	if err != nil {
		glog.Errorf("Cannot get email sent per second for tenant %s: %v", tenantID, err)
		return false
	}

	if emailPerSecond.Amount >= r.limitPerSecond && (now-emailPerSecond.UpdatedAt) < 2 {
		glog.Warningf("Reached limit for sent emails per second for tenant %s", tenantID)
		return false
	}
	if emailPerSecond.UpdatedAt == 0 || // just created EmailSentPerSecond counter
		(now-emailPerSecond.UpdatedAt) > 1 || // stale EmailSentPerSecond counter
		(emailPerSecond.Amount < r.limitPerSecond && (now-emailPerSecond.UpdatedAt) < 2) { // rate is within limit
		if err = r.dbConnection.UpdateEmailSentPerSecond(emailPerSecond.Amount + 1); err != nil {
			glog.Errorf("Cannot update email sent per second for tenant %s: %v", tenantID, err)
		}
		return true
	}

	return false
}

func (r *RateLimiterService) allowRatePerTenant(tenantID string, now time.Time) bool {
	emailPerTenant, err := r.dbConnection.GetEmailSentByTenant(tenantID, now)
	if err != nil {
		glog.Errorf("Cannot get email sent for tenant %s: %v", tenantID, err)
		return false
	}
	hoursDelta := now.Sub(emailPerTenant.Date).Hours()

	if emailPerTenant.Amount >= r.limitPerDayPerTenant && hoursDelta <= 24 {
		glog.Warningf("Reached limit for sent emails per day per tenant for tenant %s", tenantID)
		return false
	}

	if hoursDelta > 24 || (emailPerTenant.Amount < r.limitPerDayPerTenant && hoursDelta <= 24) {
		if err = r.dbConnection.UpdateEmailSentByTenant(tenantID, now, emailPerTenant.Amount+1); err != nil {
			glog.Errorf("Cannot update email sent per second for tenant %s: %v", tenantID, err)
		}
		return true
	}

	return false
}
