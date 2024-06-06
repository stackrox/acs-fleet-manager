package email

import (
	"github.com/stackrox/acs-fleet-manager/emailsender/pkg/db"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var limitPerSecond = 10
var limitPerTenant = 20
var testTenantID = "test-tenant-id"
var now = time.Now()
var staleDate = time.Date(2000, 1, 1, 1, 1, 1, 0, time.UTC)

type MockDatabaseClient struct {
	calledGetEmailSentByTenant     bool
	calledUpdateEmailSentByTenant  bool
	calledGetEmailSentPerSecond    bool
	calledUpdateEmailSentPerSecond bool

	GetEmailSentByTenantFunc     func(tenantID string, date time.Time) (*db.EmailSentByTenant, error)
	UpdateEmailSentByTenantFunc  func(tenantID string, date time.Time, amount int) error
	GetEmailSentPerSecondFunc    func() (*db.EmailSentPerSecond, error)
	UpdateEmailSentPerSecondFunc func(amount int) error
}

func (m *MockDatabaseClient) GetEmailSentByTenant(tenantID string, date time.Time) (*db.EmailSentByTenant, error) {
	m.calledGetEmailSentByTenant = true
	return m.GetEmailSentByTenantFunc(tenantID, date)
}

func (m *MockDatabaseClient) UpdateEmailSentByTenant(tenantID string, date time.Time, amount int) error {
	m.calledUpdateEmailSentByTenant = true
	return m.UpdateEmailSentByTenantFunc(tenantID, date, amount)
}

func (m *MockDatabaseClient) GetEmailSentPerSecond() (*db.EmailSentPerSecond, error) {
	m.calledGetEmailSentPerSecond = true
	return m.GetEmailSentPerSecondFunc()
}

func (m *MockDatabaseClient) UpdateEmailSentPerSecond(amount int) error {
	m.calledUpdateEmailSentPerSecond = true
	return m.UpdateEmailSentPerSecondFunc(amount)
}

func TestAllowTrue_Success(t *testing.T) {
	mockDatabaseClient := &MockDatabaseClient{
		GetEmailSentByTenantFunc: func(tenantID string, date time.Time) (*db.EmailSentByTenant, error) {
			return &db.EmailSentByTenant{TenantID: testTenantID, Date: now, Amount: 5}, nil
		},
		UpdateEmailSentByTenantFunc: func(tenantID string, date time.Time, amount int) error {
			return nil
		},
		GetEmailSentPerSecondFunc: func() (*db.EmailSentPerSecond, error) {
			return &db.EmailSentPerSecond{UpdatedAt: int(now.Unix()), Amount: 1}, nil
		},
		UpdateEmailSentPerSecondFunc: func(amount int) error {
			return nil
		},
	}

	service := RateLimiterService{
		limitPerSecond:       limitPerSecond,
		limitPerDayPerTenant: limitPerTenant,
		dbConnection:         mockDatabaseClient,
	}

	allowed := service.Allow(testTenantID)

	assert.True(t, allowed)
	assert.True(t, mockDatabaseClient.calledGetEmailSentByTenant)
	assert.True(t, mockDatabaseClient.calledUpdateEmailSentByTenant)
	assert.True(t, mockDatabaseClient.calledGetEmailSentPerSecond)
	assert.True(t, mockDatabaseClient.calledUpdateEmailSentPerSecond)
}

func TestAllowTrueOverLimitTenantButStaleDate(t *testing.T) {
	mockDatabaseClient := &MockDatabaseClient{
		GetEmailSentByTenantFunc: func(tenantID string, date time.Time) (*db.EmailSentByTenant, error) {
			return &db.EmailSentByTenant{TenantID: testTenantID, Date: staleDate, Amount: limitPerTenant + 1}, nil
		},
		UpdateEmailSentByTenantFunc: func(tenantID string, date time.Time, amount int) error {
			return nil
		},
		GetEmailSentPerSecondFunc: func() (*db.EmailSentPerSecond, error) {
			return &db.EmailSentPerSecond{UpdatedAt: int(now.Unix()), Amount: 1}, nil
		},
		UpdateEmailSentPerSecondFunc: func(amount int) error {
			return nil
		},
	}

	service := RateLimiterService{
		limitPerSecond:       limitPerSecond,
		limitPerDayPerTenant: limitPerTenant,
		dbConnection:         mockDatabaseClient,
	}

	allowed := service.Allow(testTenantID)

	assert.True(t, allowed)
	assert.True(t, mockDatabaseClient.calledGetEmailSentByTenant)
	assert.True(t, mockDatabaseClient.calledUpdateEmailSentByTenant)
	assert.True(t, mockDatabaseClient.calledGetEmailSentPerSecond)
	assert.True(t, mockDatabaseClient.calledUpdateEmailSentPerSecond)
}

func TestAllowTrueOverLimitPerSecondButStaleDate(t *testing.T) {
	mockDatabaseClient := &MockDatabaseClient{
		GetEmailSentByTenantFunc: func(tenantID string, date time.Time) (*db.EmailSentByTenant, error) {
			return &db.EmailSentByTenant{TenantID: testTenantID, Date: staleDate, Amount: 5}, nil
		},
		UpdateEmailSentByTenantFunc: func(tenantID string, date time.Time, amount int) error {
			return nil
		},
		GetEmailSentPerSecondFunc: func() (*db.EmailSentPerSecond, error) {
			return &db.EmailSentPerSecond{UpdatedAt: int(staleDate.Unix()), Amount: limitPerSecond + 1}, nil
		},
		UpdateEmailSentPerSecondFunc: func(amount int) error {
			return nil
		},
	}

	service := RateLimiterService{
		limitPerSecond:       limitPerSecond,
		limitPerDayPerTenant: limitPerTenant,
		dbConnection:         mockDatabaseClient,
	}

	allowed := service.Allow(testTenantID)

	assert.True(t, allowed)
	assert.True(t, mockDatabaseClient.calledGetEmailSentByTenant)
	assert.True(t, mockDatabaseClient.calledUpdateEmailSentByTenant)
	assert.True(t, mockDatabaseClient.calledGetEmailSentPerSecond)
	assert.True(t, mockDatabaseClient.calledUpdateEmailSentPerSecond)
}

func TestAllowFalseOverLimitPerSecond(t *testing.T) {
	mockDatabaseClient := &MockDatabaseClient{
		GetEmailSentPerSecondFunc: func() (*db.EmailSentPerSecond, error) {
			return &db.EmailSentPerSecond{UpdatedAt: int(now.Unix()), Amount: limitPerSecond + 1}, nil
		},
	}

	service := RateLimiterService{
		limitPerSecond:       limitPerSecond,
		limitPerDayPerTenant: limitPerTenant,
		dbConnection:         mockDatabaseClient,
	}

	allowed := service.Allow(testTenantID)

	assert.False(t, allowed)
	assert.False(t, mockDatabaseClient.calledGetEmailSentByTenant)
	assert.False(t, mockDatabaseClient.calledUpdateEmailSentByTenant)
	assert.True(t, mockDatabaseClient.calledGetEmailSentPerSecond)
	assert.False(t, mockDatabaseClient.calledUpdateEmailSentPerSecond)
}

func TestAllowFalseOverLimitPerTenant(t *testing.T) {
	mockDatabaseClient := &MockDatabaseClient{
		GetEmailSentByTenantFunc: func(tenantID string, date time.Time) (*db.EmailSentByTenant, error) {
			return &db.EmailSentByTenant{TenantID: testTenantID, Date: now, Amount: limitPerTenant + 1}, nil
		},
		GetEmailSentPerSecondFunc: func() (*db.EmailSentPerSecond, error) {
			return &db.EmailSentPerSecond{UpdatedAt: int(now.Unix()), Amount: 5}, nil
		},
		UpdateEmailSentPerSecondFunc: func(amount int) error {
			return nil
		},
	}

	service := RateLimiterService{
		limitPerSecond:       limitPerSecond,
		limitPerDayPerTenant: limitPerTenant,
		dbConnection:         mockDatabaseClient,
	}

	allowed := service.Allow(testTenantID)

	assert.False(t, allowed)
	assert.True(t, mockDatabaseClient.calledGetEmailSentByTenant)
	assert.False(t, mockDatabaseClient.calledUpdateEmailSentByTenant)
	assert.True(t, mockDatabaseClient.calledGetEmailSentPerSecond)
	assert.True(t, mockDatabaseClient.calledUpdateEmailSentPerSecond)
}
