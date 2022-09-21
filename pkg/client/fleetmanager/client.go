package fleetmanager

import (
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	admin "github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
)

type authTransport struct {
	transport http.RoundTripper
	auth      Auth
}

func (c *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := c.auth.AddAuth(req); err != nil {
		return nil, errors.Wrapf(err, "setting auth on req %+v", req)
	}
	resp, err := c.transport.RoundTrip(req)
	if err != nil {
		return resp, errors.Wrap(err, "round trip error")
	}
}

// newAuthTransport creates a http.RoundTripper that wraps http.DefaultTransport and injects
// the authorization header from Auth into any request.
func newAuthTransport(auth Auth) *authTransport {
	g
	return &authTransport{
		transport: http.DefaultTransport,
		auth:      auth,
	}
}

// Client is a helper struct that wraps around the API clients generated from
// OpenAPI spec for the three different API groups of fleet manager: public, private, admin.
type Client struct {
	public  *public.APIClient
	private *private.APIClient
	admin   *admin.APIClient
}

// ClientOption to configure the Client.
type ClientOption func(*options)

// WithDebugEnabled enables the debug logging for API request sent and received from fleet manager.
// Internally, this will use httputil.DumpRequestOut/DumpResponse.
func WithDebugEnabled() ClientOption {
	return func(o *options) {
		o.debug = true
	}
}

// WithUserAgent allows to set a custom value that shall be used as the User-Agent header
// when sending requests.
func WithUserAgent(userAgent string) ClientOption {
	return func(o *options) {
		o.userAgent = userAgent
	}
}

type options struct {
	debug     bool
	userAgent string
}

func defaultOptions() *options {
	return &options{
		debug:     false,
		userAgent: "OpenAPI-Generator/1.0.0/go",
	}
}

// NewClient creates a new fleet manager client with the specified auth type.
// The client will be able to talk to the three different API groups of fleet manager: public, private, admin.
func NewClient(endpoint string, auth Auth, opts ...ClientOption) (*Client, error) {
	if _, err := url.Parse(endpoint); err != nil {
		return nil, errors.Wrapf(err, "parsing endpoint %q as URL", endpoint)
	}

	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	client := &Client{}

	httpClient := &http.Client{
		Transport: newAuthTransport(auth),
	}

	client.public = public.NewAPIClient(&public.Configuration{
		BasePath:   endpoint,
		UserAgent:  o.userAgent,
		Debug:      o.debug,
		HTTPClient: httpClient,
	})
	client.private = private.NewAPIClient(&private.Configuration{
		BasePath:   endpoint,
		UserAgent:  o.userAgent,
		Debug:      o.debug,
		HTTPClient: httpClient,
	})
	client.admin = admin.NewAPIClient(&admin.Configuration{
		BasePath:   endpoint,
		UserAgent:  o.userAgent,
		Debug:      o.debug,
		HTTPClient: httpClient,
	})

	return client, nil
}

// PublicAPI returns the service to interact with fleet manager's public API.
func (c *Client) PublicAPI() *public.DefaultApiService {
	return c.public.DefaultApi
}

// PrivateAPI returns the service to interact with fleet manager's private API.
func (c *Client) PrivateAPI() *private.AgentClustersApiService {
	return c.private.AgentClustersApi
}

// AdminAPI returns the service to interact with fleet manager's admin API.
func (c *Client) AdminAPI() *admin.DefaultApiService {
	return c.admin.DefaultApi
}
