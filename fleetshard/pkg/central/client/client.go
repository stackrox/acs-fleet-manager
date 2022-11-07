// Package client ...
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/utils"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	acsErrors "github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/httputil"
)

const couldNotParseReason = "could not parse a reason for request to fail"

// reusing transport allows us to benefit from connection pooling.
var insecureTransport *http.Transport

func init() {
	insecureTransport = http.DefaultTransport.(*http.Transport).Clone()
	// TODO: ROX-11795: once certificates will be added, we probably will be able to replace with secure transport
	insecureTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
}

// Client represents the client for central.
type Client struct {
	address    string
	pass       string
	httpClient http.Client
	central    private.ManagedCentral
}

// NewCentralClient creates a new client for central with basic password authentication.
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

// NewCentralClientNoAuth creates a new client for central without authentication.
func NewCentralClientNoAuth(central private.ManagedCentral, address string) *Client {
	return &Client{
		central: central,
		address: address,
		httpClient: http.Client{
			Transport: insecureTransport,
		},
	}
}

// SendRequestToCentralRaw sends the request message to central and returns the http response.
func (c *Client) SendRequestToCentralRaw(ctx context.Context, requestMessage proto.Message, method, path string) (*http.Response, error) {
	req, err := c.createRequest(ctx, requestMessage, method, path)
	if err != nil {
		return nil, errors.Wrap(err, "creating HTTP request to central")
	}
	if c.pass != "" {
		req.SetBasicAuth("admin", c.pass)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "sending new request to central")
	}
	return resp, nil
}

// SendRequestToCentral sends the request message to central and returns the response message.
// If no response message is given, the response body will not be unmarshalled.
// It will return an error if any error occurs during request creation, unmarshalling or the request returned with a
// non-successful HTTP status code.
func (c *Client) SendRequestToCentral(ctx context.Context, requestMessage proto.Message, method, path string,
	responseMessage proto.Message) error {
	resp, err := c.SendRequestToCentralRaw(ctx, requestMessage, method, path)
	if err != nil {
		return err
	}

	defer utils.IgnoreError(resp.Body.Close)

	if !httputil.Is2xxStatusCode(resp.StatusCode) {
		reason := extractCentralError(resp)
		return acsErrors.NewErrorFromHTTPStatusCode(resp.StatusCode, "failed to execute request: %s %s with reason %q",
			method, path, reason)
	}

	// Do not try to unmarshal the response body if no response message is set.
	if responseMessage == nil {
		return nil
	}

	if err := jsonpb.Unmarshal(resp.Body, responseMessage); err != nil {
		return errors.Wrap(err, "decoding response body")
	}
	return nil
}

type centralErrorResponse struct {
	Error string `json:"error,omitempty"`
}

func extractCentralError(resp *http.Response) string {
	var data centralErrorResponse
	if resp == nil || resp.Body == nil {
		return couldNotParseReason
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return couldNotParseReason
	}
	if data.Error != "" {
		return data.Error
	}
	return couldNotParseReason
}

func (c *Client) createRequest(ctx context.Context, requestMessage proto.Message, method, path string) (*http.Request, error) {
	body := &bytes.Buffer{}
	if requestMessage != nil {
		marshaller := jsonpb.Marshaler{}
		if err := marshaller.Marshal(body, requestMessage); err != nil {
			return nil, errors.Wrap(err, "marshalling new request to central")
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.address+path, body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}
	return req, nil
}

// SendGroupRequest sends a request to create the specified group.
// It will return an error if any error occurs during request creation or the request returned with a non-successful
// HTTP status code.
func (c *Client) SendGroupRequest(ctx context.Context, groupRequest *storage.Group) error {
	if err := c.SendRequestToCentral(ctx, groupRequest, http.MethodPost, "/v1/groups",
		nil); err != nil {
		return errors.Wrapf(err, "failed to create group for central %s/%s",
			c.central.Metadata.Namespace, c.central.Metadata.Name)
	}
	return nil
}

// SendAuthProviderRequest sends a request to create the specified auth provider and returns the created auth provider.
// It will return an error if any error occurs during request creation or the request returned with a non-successful
// HTTP status code.
func (c *Client) SendAuthProviderRequest(ctx context.Context, authProviderRequest *storage.AuthProvider) (*storage.AuthProvider, error) {
	var authProviderResponse storage.AuthProvider
	if err := c.SendRequestToCentral(ctx, authProviderRequest, http.MethodPost, "/v1/authProviders",
		&authProviderResponse); err != nil {
		return nil, errors.Wrapf(err, "failed to create auth provider for central %s/%s",
			c.central.Metadata.Namespace, c.central.Metadata.Name)
	}
	return &authProviderResponse, nil
}

// GetLoginAuthProviders sends a request to retrieve all login auth providers and returns them.
// It will return an error if any error occurs during request creation or the request returned with a non-successful
// HTTP status code.
func (c *Client) GetLoginAuthProviders(ctx context.Context) (*v1.GetLoginAuthProvidersResponse, error) {
	var loginAuthProvidersResponse v1.GetLoginAuthProvidersResponse
	if err := c.SendRequestToCentral(ctx, nil, http.MethodGet, "/v1/login/authproviders",
		&loginAuthProvidersResponse); err != nil {
		return nil, errors.Wrapf(err, "failed to get login auth providers from central %s/%s",
			c.central.Metadata.Namespace, c.central.Metadata.Name)
	}
	return &loginAuthProvidersResponse, nil
}
