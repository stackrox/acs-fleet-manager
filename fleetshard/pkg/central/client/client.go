package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
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

type Client struct {
	address    string
	pass       string
	httpClient http.Client
	central    private.ManagedCentral
}

type AuthProviderResponse struct {
	Id string `json:"id"`
}

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

func (c *Client) SendRequestToCentral(ctx context.Context, requestBody interface{}, path string) (*http.Response, error) {
	jsonBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, errors.Wrap(err, "marshalling new request to central")
	}
	req, err := http.NewRequest(http.MethodPost, c.address+path, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, errors.Wrap(err, "creating HTTP request to central")
	}
	req.SetBasicAuth("admin", c.pass)
	req = req.WithContext(ctx)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "sending new request to central")
	}
	return resp, nil
}

func (c *Client) SendGroupRequest(ctx context.Context, groupRequest *storage.Group) error {
	resp, err := c.SendRequestToCentral(ctx, groupRequest, "/v1/groups")
	if err != nil {
		return errors.Wrap(err, "sending new group to central")
	}
	if !httputil.Is2xxStatusCode(resp.StatusCode) {
		return acsErrors.NewErrorFromHTTPStatusCode(resp.StatusCode, "failed to create group for central %s/%s", c.central.Metadata.Namespace, c.central.Metadata.Name)
	}
	return nil
}

func (c *Client) SendAuthProviderRequest(ctx context.Context, authProviderRequest *storage.AuthProvider) (*AuthProviderResponse, error) {
	resp, err := c.SendRequestToCentral(ctx, authProviderRequest, "/v1/authProviders")
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
