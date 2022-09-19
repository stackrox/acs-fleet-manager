package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/compat"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
)

// Client represents the REST client for connecting to fleet-manager
type Client struct {
	client           http.Client
	auth             fleetmanager.Auth
	adminAPIEndpoint string
}

func (c *Client) newRequest(method string, url string, body io.Reader) (*http.Response, error) {
	r, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("building HTTP request: %w", err)
	}
	if err := c.auth.AddAuth(r); err != nil {
		return nil, fmt.Errorf("adding authentication information to HTTP request: %w", err)
	}

	resp, err := c.client.Do(r)
	if err != nil {
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
		return fmt.Errorf("reading HTTP response body: %w", err)
	}
	if len(data) == 0 {
		return nil
	}

	into := struct {
		Kind string `json:"kind"`
	}{}
	err = json.Unmarshal(data, &into)
	if err != nil {
		return fmt.Errorf("extracting kind information from HTTP response: %w", err)
	}

	// Unmarshal error
	if into.Kind == "Error" || into.Kind == "error" {
		apiError := compat.Error{}
		err = json.Unmarshal(data, &apiError)
		if err != nil {
			return fmt.Errorf("unmarshalling HTTP response as error: %w", err)
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

// NewAdminClient ...
func NewAdminClient(uriBase string, auth fleetmanager.Auth) (*Client, error) {
	return &Client{
		client:           http.Client{},
		auth:             auth,
		adminAPIEndpoint: fmt.Sprintf("%s/%s", uriBase, "api/rhacs/v1/admin"),
	}, nil
}

// CreateCentral creates a central using fleet-manager's private Admin API.
func (c *Client) CreateCentral(request public.CentralRequestPayload) (*public.CentralRequest, error) {
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshalling HTTP request: %w", err)
	}

	resp, err := c.newRequest(http.MethodPost, fmt.Sprintf("%s/dinosaurs?async=true", c.adminAPIEndpoint), bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("executing HTTP request: %w", err)
	}

	result := &public.CentralRequest{}
	err = c.unmarshalResponse(resp, result)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling HTTP response: %w", err)
	}
	return result, nil
}

// UpdateCentral updates a central using fleet-manager's private Admin API.
func (c *Client) UpdateCentral(id string, updateReq private.CentralUpdateRequest) (*private.Central, error) {
	reqBody, err := json.Marshal(updateReq)
	if err != nil {
		return nil, fmt.Errorf("marshalling HTTP request: %w", err)
	}

	resp, err := c.newRequest(http.MethodPatch, fmt.Sprintf("%s/dinosaurs/%s", c.adminAPIEndpoint, id), bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("executing HTTP request: %w", err)
	}

	result := &private.Central{}
	err = c.unmarshalResponse(resp, result)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling HTTP response: %w", err)
	}
	return result, nil
}
