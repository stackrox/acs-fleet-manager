package fleetmanager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetshardmetrics"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/compat"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
)

const (
	uri         = "api/rhacs/v1/agent-clusters"
	statusRoute = "status"

	publicCentralURI = "api/rhacs/v1/centrals"
)

//go:generate moq -out client_mock.go . FleetManagerClient
type FleetManagerClient interface {
	GetManagedCentralList() (*private.ManagedCentralList, error)
	UpdateStatus(statuses map[string]private.DataPlaneCentralStatus) error
	CreateCentral(request public.CentralRequestPayload) (*public.CentralRequest, error)
	GetCentral(id string) (*public.CentralRequest, error)
	DeleteCentral(id string) error
}

var _ FleetManagerClient = (*Client)(nil)

// Client represents the REST client for connecting to fleet-manager
type Client struct {
	client                http.Client
	auth                  Auth
	clusterID             string
	fleetshardAPIEndpoint string
	consoleAPIEndpoint    string
}

// NewClient creates a new client
func NewClient(endpoint string, clusterID string, auth Auth) (FleetManagerClient, error) {
	if clusterID == "" {
		return nil, errors.New("cluster id is empty")
	}

	if endpoint == "" {
		return nil, errors.New("fleetshardAPIEndpoint is empty")
	}

	return &Client{
		client:                http.Client{},
		clusterID:             clusterID,
		auth:                  auth,
		fleetshardAPIEndpoint: fmt.Sprintf("%s/%s/%s/%s", endpoint, uri, clusterID, "centrals"),
		consoleAPIEndpoint:    fmt.Sprintf("%s/%s", endpoint, publicCentralURI),
	}, nil
}

// GetManagedCentralList returns a list of centrals from fleet-manager which should be managed by this fleetshard.
func (c *Client) GetManagedCentralList() (*private.ManagedCentralList, error) {
	resp, err := c.newRequest(http.MethodGet, c.fleetshardAPIEndpoint, &bytes.Buffer{})
	if err != nil {
		return nil, err
	}

	list := &private.ManagedCentralList{}
	err = c.unmarshalResponse(resp, &list)
	if err != nil {
		return nil, errors.Wrapf(err, "calling %s", c.fleetshardAPIEndpoint)
	}

	return list, nil
}

// UpdateStatus batch updates the status of managed centrals. The status param takes a map of DataPlaneCentralStatus indexed by
// the Centrals ID.
func (c *Client) UpdateStatus(statuses map[string]private.DataPlaneCentralStatus) error {
	updateBody, err := json.Marshal(statuses)
	if err != nil {
		return fmt.Errorf("marshalling data-plane central status: %w", err)
	}

	resp, err := c.newRequest(http.MethodPut, fmt.Sprintf("%s/%s", c.fleetshardAPIEndpoint, statusRoute), bytes.NewBuffer(updateBody))
	if err != nil {
		return err
	}

	if err := c.unmarshalResponse(resp, nil); err != nil {
		return errors.Wrapf(err, "updating status")
	}
	return nil
}

// CreateCentral creates a central from the public fleet-manager API
func (c *Client) CreateCentral(request public.CentralRequestPayload) (*public.CentralRequest, error) {
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshalling request for central creation: %w", err)
	}

	resp, err := c.newRequest(http.MethodPost, fmt.Sprintf("%s?async=true", c.consoleAPIEndpoint), bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	result := &public.CentralRequest{}
	err = c.unmarshalResponse(resp, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetCentral returns a Central from the public fleet-manager API
func (c *Client) GetCentral(id string) (*public.CentralRequest, error) {
	resp, err := c.newRequest(http.MethodGet, fmt.Sprintf("%s/%s", c.consoleAPIEndpoint, id), nil)
	if err != nil {
		return nil, err
	}

	central := &public.CentralRequest{}
	err = c.unmarshalResponse(resp, central)
	if err != nil {
		return nil, err
	}

	return central, nil
}

// DeleteCentral deletes a central from the public fleet-manager API
func (c *Client) DeleteCentral(id string) error {
	resp, err := c.newRequest(http.MethodDelete, fmt.Sprintf("%s/%s?async=true", c.consoleAPIEndpoint, id), nil)
	if err != nil {
		return err
	}

	err = c.unmarshalResponse(resp, nil)
	if err != nil {
		return errors.Wrapf(err, "deleting central %s", id)
	}
	return nil
}

func (c *Client) newRequest(method string, url string, body io.Reader) (*http.Response, error) {
	glog.Infof("Send request to %s", url)
	r, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}
	if err := c.auth.AddAuth(r); err != nil {
		return nil, fmt.Errorf("adding authentication information to request: %w", err)
	}

	fleetshardmetrics.MetricsInstance().IncrementFleetManagerRequests()
	resp, err := c.client.Do(r)
	if err != nil {
		fleetshardmetrics.MetricsInstance().IncrementFleetManagerRequestErrors()
		return nil, fmt.Errorf("executing HTTP request: %w", err)
	}
	return resp, nil
}

// unmarshalResponse unmarshalls a fleet-manager response. It returns an error if
// fleet-manager returns errors from its API.
// If the value v is nil the response is not marshalled into a struct, instead only checked for an API error.
func (c *Client) unmarshalResponse(resp *http.Response, v interface{}) error {
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}
	if len(data) == 0 {
		return nil
	}

	into := struct {
		Kind string `json:"kind"`
	}{}
	err = json.Unmarshal(data, &into)
	if err != nil {
		return fmt.Errorf("retrieving 'kind' property from HTTP response: %w", err)
	}

	// Unmarshal error
	if into.Kind == "Error" || into.Kind == "error" {
		fleetshardmetrics.MetricsInstance().IncrementFleetManagerRequestErrors()
		apiError := compat.Error{}
		err = json.Unmarshal(data, &apiError)
		if err != nil {
			return fmt.Errorf("unmarshalling error HTTP response: %w", err)
		}
		return errors.Errorf("API error (HTTP status %d) occured %s: %s", resp.StatusCode, apiError.Code, apiError.Reason)
	}

	if v == nil {
		return nil
	}

	err = json.Unmarshal(data, v)
	if err != nil {
		return fmt.Errorf("unmarshalling HTTP response as %T: %w", v, err)
	}
	return nil
}
