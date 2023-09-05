package fleetmanager

import (
	"context"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	admin "github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
)

//go:generate moq -out api_moq.go . PublicAPI PrivateAPI AdminAPI

// PublicAPI is a wrapper interface for the fleetmanager client public API.
type PublicAPI interface {
	CreateCentral(ctx context.Context, async bool, request public.CentralRequestPayload) (public.CentralRequest, *http.Response, error)
	DeleteCentralById(ctx context.Context, id string, async bool) (*http.Response, error)
	GetCentralById(ctx context.Context, id string) (public.CentralRequest, *http.Response, error)
	GetCentrals(ctx context.Context, localVarOptionals *public.GetCentralsOpts) (public.CentralRequestList, *http.Response, error)
}

// PrivateAPI is a wrapper interface for the fleetmanager client private API.
type PrivateAPI interface {
	GetDataPlaneClusterAgentConfig(ctx context.Context, id string) (private.DataplaneClusterAgentConfig, *http.Response, error)
	GetCentral(ctx context.Context, centralID string) (private.ManagedCentral, *http.Response, error)
	GetCentrals(ctx context.Context, id string) (private.ManagedCentralList, *http.Response, error)
	UpdateCentralClusterStatus(ctx context.Context, id string, requestBody map[string]private.DataPlaneCentralStatus) (*http.Response, error)
}

// AdminAPI is a wrapper interface for the fleetmanager client admin API.
type AdminAPI interface {
	GetCentrals(ctx context.Context, localVarOptionals *admin.GetCentralsOpts) (admin.CentralList, *http.Response, error)
	CreateCentral(ctx context.Context, async bool, centralRequestPayload admin.CentralRequestPayload) (admin.CentralRequest, *http.Response, error)
	UpdateCentralById(ctx context.Context, id string, centralUpdateRequest admin.CentralUpdateRequest) (admin.Central, *http.Response, error)
	DeleteDbCentralById(ctx context.Context, id string) (*http.Response, error)
}

var (
	_ http.RoundTripper = (*authTransport)(nil)
	_ PublicAPI         = (*publicAPIDelegate)(nil)
	_ PrivateAPI        = (*privateAPIDelegate)(nil)
	_ AdminAPI          = (*adminAPIDelegate)(nil)
)

type publicAPIDelegate struct {
	*public.DefaultApiService
}

type privateAPIDelegate struct {
	*private.AgentClustersApiService
}

type adminAPIDelegate struct {
	*admin.DefaultApiService
}

type authTransport struct {
	transport http.RoundTripper
	auth      Auth
}

func (c *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := c.auth.AddAuth(req); err != nil {
		return nil, errors.Wrapf(err, "setting auth on req %+v", req)
	}
	return c.transport.RoundTrip(req)
}

// newAuthTransport creates a http.RoundTripper that wraps http.DefaultTransport and injects
// the authorization header from Auth into any request.
func newAuthTransport(auth Auth) *authTransport {
	return &authTransport{
		transport: http.DefaultTransport,
		auth:      auth,
	}
}

// Client is a helper struct that wraps around the API clients generated from
// OpenAPI spec for the three different API groups of fleet manager: public, private, admin.
type Client struct {
	publicAPI  PublicAPI
	privateAPI PrivateAPI
	adminAPI   AdminAPI
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

	client.publicAPI = &publicAPIDelegate{
		DefaultApiService: public.NewAPIClient(&public.Configuration{
			BasePath:   endpoint,
			UserAgent:  o.userAgent,
			Debug:      o.debug,
			HTTPClient: httpClient,
		}).DefaultApi,
	}
	client.privateAPI = &privateAPIDelegate{
		AgentClustersApiService: private.NewAPIClient(&private.Configuration{
			BasePath:   endpoint,
			UserAgent:  o.userAgent,
			Debug:      o.debug,
			HTTPClient: httpClient,
		}).AgentClustersApi,
	}
	client.adminAPI = &adminAPIDelegate{
		DefaultApiService: admin.NewAPIClient(&admin.Configuration{
			BasePath:   endpoint,
			UserAgent:  o.userAgent,
			Debug:      o.debug,
			HTTPClient: httpClient,
		}).DefaultApi,
	}

	return client, nil
}

// PublicAPI returns the service to interact with fleet manager's public API.
func (c *Client) PublicAPI() PublicAPI {
	return c.publicAPI
}

// PrivateAPI returns the service to interact with fleet manager's private API.
func (c *Client) PrivateAPI() PrivateAPI {
	return c.privateAPI
}

// AdminAPI returns the service to interact with fleet manager's admin API.
func (c *Client) AdminAPI() AdminAPI {
	return c.adminAPI
}
