package fleetmanager

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
func (m *ClientMock) Client() *Client {
	return &Client{
		privateAPI: m.PrivateAPIMock,
		publicAPI:  m.PublicAPIMock,
		adminAPI:   m.AdminAPIMock,
	}
}
