package email

import (
	"testing"
	"time"

	"github.com/stackrox/acs-fleet-manager/emailsender/pkg/db"
	"github.com/stretchr/testify/assert"
)

var limitPerTenant = 20
var testTenantID = "test-tenant-id"

func TestAllowTrue_Success(t *testing.T) {
	mockDatabaseClient := &db.MockDatabaseClient{
		CountEmailSentByTenantFromFunc: func(tenantID string, from time.Time) (int64, error) {
			return int64(limitPerTenant - 1), nil
		},
	}

	service := RateLimiterService{
		limitPerTenant: limitPerTenant,
		dbConnection:   mockDatabaseClient,
	}

	allowed, err := service.IsAllowed(testTenantID)

	assert.True(t, allowed)
	assert.Nil(t, err)
	assert.True(t, mockDatabaseClient.CalledCountEmailSentByTenantFrom)
}

func TestAllowFalse_LimitReached(t *testing.T) {
	mockDatabaseClient := &db.MockDatabaseClient{
		CountEmailSentByTenantFromFunc: func(tenantID string, from time.Time) (int64, error) {
			return int64(limitPerTenant + 1), nil
		},
	}

	service := RateLimiterService{
		limitPerTenant: limitPerTenant,
		dbConnection:   mockDatabaseClient,
	}

	allowed, err := service.IsAllowed(testTenantID)

	assert.False(t, allowed)
	assert.Nil(t, err)
	assert.True(t, mockDatabaseClient.CalledCountEmailSentByTenantFrom)
}

func TestPersistEmailSendEvent(t *testing.T) {
	mockDatabaseClient := &db.MockDatabaseClient{
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
	assert.True(t, mockDatabaseClient.CalledInsertEmailSentByTenant)
}
