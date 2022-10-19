package fleetmanager

import (
	"context"
	"net/http"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
)

// CreateCentralResponse is returned by the client for the central creation endpoint.
type CreateCentralResponse struct {
	Request  public.CentralRequest
	Response *http.Response
	Err      error
}

// DeleteCentralByIDResponse is returned by the client for the central deletion endpoint.
type DeleteCentralByIDResponse struct {
	Response *http.Response
	Err      error
}

// GetCentralByIDResponse is returned by the client for the central get endpoint.
type GetCentralByIDResponse struct {
	Request  public.CentralRequest
	Response *http.Response
	Err      error
}

// MockClient is a fake client used for unit testing against fleet manager.
type MockClient struct {
	createCentralResponses     []*CreateCentralResponse
	deleteCentralByIDResponses []*DeleteCentralByIDResponse
	getCentralByIDResponses    []*GetCentralByIDResponse
}

// NewMock returns a new mock client.
func NewMock() (*MockClient, error) {
	return &MockClient{}, nil
}

// AddCreateCentralResponse adds a response to the CreateCentral endpoint.
func (c *MockClient) AddCreateCentralResponse(response *CreateCentralResponse) {
	c.createCentralResponses = append(c.createCentralResponses, response)
}

// AddDeleteCentralByIDResponse adds a response to the DeleteCentralById endpoint.
func (c *MockClient) AddDeleteCentralByIDResponse(response *DeleteCentralByIDResponse) {
	c.deleteCentralByIDResponses = append(c.deleteCentralByIDResponses, response)
}

// AddGetCentralByIDResponse adds a response to the GetCentralByIdResponse endpoint.
func (c *MockClient) AddGetCentralByIDResponse(response *GetCentralByIDResponse) {
	c.getCentralByIDResponses = append(c.getCentralByIDResponses, response)
}

// ClearResponses deletes all stored responses.
func (c *MockClient) ClearResponses(response *GetCentralByIDResponse) {
	c.createCentralResponses = nil
	c.deleteCentralByIDResponses = nil
	c.getCentralByIDResponses = nil
}

// CreateCentral mocks a Central creation request.
func (c *MockClient) CreateCentral(ctx context.Context, async bool, payload public.CentralRequestPayload) (public.CentralRequest, *http.Response, error) {
	if len(c.createCentralResponses) == 0 {
		return public.CentralRequest{}, nil, nil
	}
	request := c.createCentralResponses[0].Request
	response := c.createCentralResponses[0].Response
	err := c.createCentralResponses[0].Err

	if len(c.createCentralResponses) > 0 {
		c.createCentralResponses = c.createCentralResponses[1:]
	}
	return request, response, err
}

// DeleteCentralById mocks a Central deletion request.
//nolint:revive
func (c *MockClient) DeleteCentralById(ctx context.Context, id string, async bool) (*http.Response, error) {
	if len(c.deleteCentralByIDResponses) == 0 {
		return nil, nil
	}
	response := c.deleteCentralByIDResponses[0].Response
	err := c.deleteCentralByIDResponses[0].Err

	if len(c.deleteCentralByIDResponses) > 0 {
		c.deleteCentralByIDResponses = c.deleteCentralByIDResponses[1:]
	}
	return response, err
}

// GetCentralById mocks a Central get request.
//nolint:revive
func (c *MockClient) GetCentralById(ctx context.Context, id string) (public.CentralRequest, *http.Response, error) {
	if len(c.getCentralByIDResponses) == 0 {
		return public.CentralRequest{}, nil, nil
	}
	request := c.getCentralByIDResponses[0].Request
	response := c.getCentralByIDResponses[0].Response
	err := c.getCentralByIDResponses[0].Err

	if len(c.getCentralByIDResponses) > 0 {
		c.getCentralByIDResponses = c.getCentralByIDResponses[1:]
	}
	return request, response, err
}
