package impl

import (
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	admin "github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/public"
	fleetmanager "github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
)

var (
	_ http.RoundTripper       = (*authTransport)(nil)
	_ fleetmanager.PublicAPI  = (*publicAPIDelegate)(nil)
	_ fleetmanager.PrivateAPI = (*privateAPIDelegate)(nil)
	_ fleetmanager.AdminAPI   = (*adminAPIDelegate)(nil)
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

// ClientOption to configure the Client.
type ClientOption func(*options)

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
func NewClient(endpoint string, auth Auth, opts ...ClientOption) (*fleetmanager.Client, error) {
	if _, err := url.Parse(endpoint); err != nil {
		return nil, errors.Wrapf(err, "parsing endpoint %q as URL", endpoint)
	}

	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	httpClient := &http.Client{
		Transport: newAuthTransport(auth),
	}

	publicAPI := &publicAPIDelegate{
		DefaultApiService: public.NewAPIClient(&public.Configuration{
			BasePath:   endpoint,
			UserAgent:  o.userAgent,
			Debug:      o.debug,
			HTTPClient: httpClient,
		}).DefaultApi,
	}
	privateAPI := &privateAPIDelegate{
		AgentClustersApiService: private.NewAPIClient(&private.Configuration{
			BasePath:   endpoint,
			UserAgent:  o.userAgent,
			Debug:      o.debug,
			HTTPClient: httpClient,
		}).AgentClustersApi,
	}
	adminAPI := &adminAPIDelegate{
		DefaultApiService: admin.NewAPIClient(&admin.Configuration{
			BasePath:   endpoint,
			UserAgent:  o.userAgent,
			Debug:      o.debug,
			HTTPClient: httpClient,
		}).DefaultApi,
	}

	return fleetmanager.MakeClient(publicAPI, privateAPI, adminAPI), nil
}
