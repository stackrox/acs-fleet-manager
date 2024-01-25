package fleetmanager

import (
	mocks "github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager/mocks"
)

// ClientMock API mocks holder.
type ClientMock struct {
	PublicAPIMock  *mocks.PublicAPIMock
	PrivateAPIMock *mocks.PrivateAPIMock
	AdminAPIMock   *mocks.AdminAPIMock
}

// NewClientMock creates a new instance of ClientMock
func NewClientMock() *ClientMock {
	return &ClientMock{
		PublicAPIMock:  &mocks.PublicAPIMock{},
		PrivateAPIMock: &mocks.PrivateAPIMock{},
		AdminAPIMock:   &mocks.AdminAPIMock{},
	}
}

// Client returns new Client instance
func (m *ClientMock) Client() *Client {
	return &Client{
		privateAPI: m.PrivateAPIMock,
		publicAPI:  m.PublicAPIMock,
		adminAPI:   m.AdminAPIMock,
	}
}
