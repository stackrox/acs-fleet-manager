// Package client ...
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"net/http"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
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
