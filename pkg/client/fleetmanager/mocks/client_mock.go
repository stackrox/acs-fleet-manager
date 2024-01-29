package mocks

import (
	fleetmanager "github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
)

// ClientMock API mocks holder.
type ClientMock struct {
	PublicAPIMock  *PublicAPIMock
	PrivateAPIMock *PrivateAPIMock
	AdminAPIMock   *AdminAPIMock
}

// NewClientMock creates a new instance of ClientMock
func NewClientMock() *ClientMock {
	return &ClientMock{
		PublicAPIMock:  &PublicAPIMock{},
		PrivateAPIMock: &PrivateAPIMock{},
		AdminAPIMock:   &AdminAPIMock{},
	}
}

// Client returns new Client instance
func (m *ClientMock) Client() *fleetmanager.Client {
	return fleetmanager.MakeClient(m.PublicAPIMock, m.PrivateAPIMock, m.AdminAPIMock)
}
