package email

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var limitPerTenant = 20
var testTenantID = "test-tenant-id"

type MockDatabaseClient struct {
	calledInsertEmailSentByTenant    bool
	calledCountEmailSentByTenantFrom bool

	InsertEmailSentByTenantFunc    func(tenantID string) error
	CountEmailSentByTenantFromFunc func(tenantID string, from time.Time) (int64, error)
}

func (m *MockDatabaseClient) InsertEmailSentByTenant(tenantID string) error {
	m.calledInsertEmailSentByTenant = true
	return m.InsertEmailSentByTenantFunc(tenantID)
}

func (m *MockDatabaseClient) CountEmailSentByTenantSince(tenantID string, from time.Time) (int64, error) {
	m.calledCountEmailSentByTenantFrom = true
	return m.CountEmailSentByTenantFromFunc(tenantID, from)
}

func TestAllowTrue_Success(t *testing.T) {
	mockDatabaseClient := &MockDatabaseClient{
		CountEmailSentByTenantFromFunc: func(tenantID string, from time.Time) (int64, error) {
			return int64(limitPerTenant - 1), nil
		},
	}

	service := RateLimiterService{
		limitPerTenant: limitPerTenant,
		dbConnection:   mockDatabaseClient,
	}

	allowed := service.IsAllowed(testTenantID)

	assert.True(t, allowed)
	assert.True(t, mockDatabaseClient.calledCountEmailSentByTenantFrom)
}

func TestAllowFalse_LimitReached(t *testing.T) {
	mockDatabaseClient := &MockDatabaseClient{
		CountEmailSentByTenantFromFunc: func(tenantID string, from time.Time) (int64, error) {
			return int64(limitPerTenant + 1), nil
		},
	}

	service := RateLimiterService{
		limitPerTenant: limitPerTenant,
		dbConnection:   mockDatabaseClient,
	}

	allowed := service.IsAllowed(testTenantID)

	assert.False(t, allowed)
	assert.True(t, mockDatabaseClient.calledCountEmailSentByTenantFrom)
}

func TestPersistEmailSendEvent(t *testing.T) {
	mockDatabaseClient := &MockDatabaseClient{
		InsertEmailSentByTenantFunc: func(tenantID string) error {
			return nil
		},
	}

	service := RateLimiterService{
		limitPerTenant: limitPerTenant,
		dbConnection:   mockDatabaseClient,
	}

	err := service.PersistEmailSendEvent(testTenantID)

	assert.NoError(t, err)
	assert.True(t, mockDatabaseClient.calledInsertEmailSentByTenant)
}
