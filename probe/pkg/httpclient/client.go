package httpclient

import (
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/stackrox/acs-fleet-manager/probe/config"
)

// New creates a http.Client with pre-configured retry and timeout.
func New(config *config.Config) *http.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.HTTPClient.Timeout = config.ProbeHttpRequestTimeout
	return retryClient.StandardClient()
}
