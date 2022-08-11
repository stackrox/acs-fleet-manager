package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	acsErrors "github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/httputil"
)

// reusing transport allows us to benefit from connection pooling
var insecureTransport *http.Transport

func init() {
	insecureTransport = http.DefaultTransport.(*http.Transport).Clone()
	// TODO: ROX-11795: once certificates will be added, we probably will be able to replace with secure transport
	insecureTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
}

// Client ...
type Client struct {
	address    string
	pass       string
	httpClient http.Client
	central    private.ManagedCentral
}

// AuthProviderResponse ...
type AuthProviderResponse struct {
	ID string `json:"id"`
}

// GetLoginAuthProviderResponse ...
type GetLoginAuthProviderResponse struct {
	AuthProviders []*GetLoginAuthProviderResponseAuthProvider `json:"authProviders"`
}

// GetLoginAuthProviderResponseAuthProvider ...
type GetLoginAuthProviderResponseAuthProvider struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

// NewCentralClient ...
func NewCentralClient(central private.ManagedCentral, address, pass string) *Client {
	return &Client{
		central: central,
		address: address,
		pass:    pass,
		httpClient: http.Client{
			Transport: insecureTransport,
		},
	}
}

// NewCentralClientNoAuth ...
func NewCentralClientNoAuth(central private.ManagedCentral, address string) *Client {
	return &Client{
		central: central,
		address: address,
		httpClient: http.Client{
			Transport: insecureTransport,
		},
	}
}

// SendRequestToCentral ...
func (c *Client) SendRequestToCentral(ctx context.Context, requestBody interface{}, method, path string) (*http.Response, error) {
	req, err := c.createRequest(requestBody, method, path)
	if err != nil {
		return nil, errors.Wrap(err, "creating HTTP request to central")
	}
	if c.pass != "" {
		req.SetBasicAuth("admin", c.pass)
	}
	req = req.WithContext(ctx)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "sending new request to central")
	}
	return resp, nil
}

func (c *Client) createRequest(requestBody interface{}, method, path string) (*http.Request, error) {
	var body io.Reader
	if requestBody != nil {
		jsonBytes, err := json.Marshal(requestBody)
		if err != nil {
			return nil, errors.Wrap(err, "marshalling new request to central")
		}
		body = bytes.NewReader(jsonBytes)
	}
	req, err := http.NewRequest(method, c.address+path, body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}
	return req, nil
}

// SendGroupRequest ...
func (c *Client) SendGroupRequest(ctx context.Context, groupRequest *storage.Group) error {
	resp, err := c.SendRequestToCentral(ctx, groupRequest, http.MethodPost, "/v1/groups")
	if err != nil {
		return errors.Wrap(err, "sending new group to central")
	}
	if !httputil.Is2xxStatusCode(resp.StatusCode) {
		return acsErrors.NewErrorFromHTTPStatusCode(resp.StatusCode, "failed to create group for central %s/%s", c.central.Metadata.Namespace, c.central.Metadata.Name)
	}
	return nil
}

// SendAuthProviderRequest ...
func (c *Client) SendAuthProviderRequest(ctx context.Context, authProviderRequest *storage.AuthProvider) (*AuthProviderResponse, error) {
	resp, err := c.SendRequestToCentral(ctx, authProviderRequest, http.MethodPost, "/v1/authProviders")
	if err != nil {
		return nil, errors.Wrap(err, "sending new auth provider to central")
	} else if !httputil.Is2xxStatusCode(resp.StatusCode) {
		return nil, acsErrors.NewErrorFromHTTPStatusCode(resp.StatusCode, "failed to create auth provider for central %s/%s", c.central.Metadata.Namespace, c.central.Metadata.Name)
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			glog.Warningf("Attempt to close auth provider response failed: %s", err)
		}
	}()
	var authProviderResp AuthProviderResponse
	err = json.NewDecoder(resp.Body).Decode(&authProviderResp)
	if err != nil {
		return nil, errors.Wrap(err, "decoding auth provider POST response")
	}
	return &authProviderResp, nil
}

// GetLoginAuthProviders ...
func (c *Client) GetLoginAuthProviders(ctx context.Context) (*GetLoginAuthProviderResponse, error) {
	resp, err := c.SendRequestToCentral(ctx, nil, http.MethodGet, "/v1/login/authproviders")
	if err != nil {
		return nil, errors.Wrap(err, "sending GetLoginAuthProviders request to central")
	} else if !httputil.Is2xxStatusCode(resp.StatusCode) {
		return nil, acsErrors.NewErrorFromHTTPStatusCode(resp.StatusCode, "failed to GetLoginAuthProviders from central %s/%s", c.central.Metadata.Namespace, c.central.Metadata.Name)
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			glog.Warningf("Attempt to close GetLoginAuthProviders response failed: %s", err)
		}
	}()
	var authProvidersResp GetLoginAuthProviderResponse
	err = json.NewDecoder(resp.Body).Decode(&authProvidersResp)
	if err != nil {
		return nil, errors.Wrap(err, "decoding GetLoginAuthProviders response")
	}
	return &authProvidersResp, nil
}
