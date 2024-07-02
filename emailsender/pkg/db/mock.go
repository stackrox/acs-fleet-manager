package db

import "time"

type MockDatabaseClient struct {
	CalledInsertEmailSentByTenant    bool
	CalledCountEmailSentByTenantFrom bool
	CalledCleanupEmailSentByTenant   bool

	InsertEmailSentByTenantFunc    func(tenantID string) error
	CountEmailSentByTenantFromFunc func(tenantID string, from time.Time) (int64, error)
	CleanupEmailSentByTenantFunc   func(before time.Time) (int64, error)
}

func (m *MockDatabaseClient) InsertEmailSentByTenant(tenantID string) error {
	m.CalledInsertEmailSentByTenant = true
	return m.InsertEmailSentByTenantFunc(tenantID)
}

func (m *MockDatabaseClient) CountEmailSentByTenantSince(tenantID string, from time.Time) (int64, error) {
	m.CalledCountEmailSentByTenantFrom = true
	return m.CountEmailSentByTenantFromFunc(tenantID, from)
}

func (m *MockDatabaseClient) CleanupEmailSentByTenant(before time.Time) (int64, error) {
	m.CalledCleanupEmailSentByTenant = true
	return m.CleanupEmailSentByTenantFunc(before)
}
