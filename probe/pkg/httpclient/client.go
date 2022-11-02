package httpclient

import (
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/stackrox/acs-fleet-manager/probe/config"
)

// New creates a http.Client with pre-configured retry and timeout.
func New(config *config.Config) *http.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.HTTPClient.Timeout = config.ProbeHttpRequestTimeout
	retryClient.RetryMax = 4
	retryClient.RetryWaitMax = 30 * time.Second
	retryClient.RetryWaitMin = 1 * time.Second
	return retryClient.StandardClient()
}
