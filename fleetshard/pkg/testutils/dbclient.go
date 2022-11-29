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
func (m *DBProvisioningClientMock) EnsureDBProvisioned(ctx context.Context, centralNamespace, centralDbSecretName string) (string, error) {
	args := m.Called(ctx, centralNamespace, centralDbSecretName)
	return args.String(0), args.Error(1)
}

// EnsureDBDeprovisioned is a mock for cloudprovider.DBClient.EnsureDBDeprovisioned
func (m *DBProvisioningClientMock) EnsureDBDeprovisioned(centralNamespace string) (bool, error) {
	args := m.Called(centralNamespace)
	return args.Bool(0), args.Error(1)
}
