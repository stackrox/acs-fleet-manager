package testutils

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// DBProvisioningClientMock is a mock cloudprovider.DBClient
type DBProvisioningClientMock struct {
	mock.Mock
}

// EnsureDBProvisioned is a mock for cloudprovider.DBClient.EnsureDBProvisioned
func (m *DBProvisioningClientMock) EnsureDBProvisioned(ctx context.Context, databaseID, masterPassword string) (string, error) {
	args := m.Called(ctx, databaseID, masterPassword)
	return args.String(0), args.Error(1)
}

// EnsureDBDeprovisioned is a mock for cloudprovider.DBClient.EnsureDBDeprovisioned
func (m *DBProvisioningClientMock) EnsureDBDeprovisioned(databaseID string) (bool, error) {
	args := m.Called(databaseID)
	return args.Bool(0), args.Error(1)
}
