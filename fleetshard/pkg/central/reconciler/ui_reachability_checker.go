package reconciler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

const (
	httpCheckTimeout = 10 * time.Second
)

// CentralUIReachabilityChecker checks if a Central UI is reachable
type CentralUIReachabilityChecker interface {
	IsCentralUIHostReachable(ctx context.Context, uiHost string) (bool, error)
}

// HTTPCentralUIReachabilityChecker is the default implementation that performs actual HTTP checks
type HTTPCentralUIReachabilityChecker struct {
	httpClient *http.Client
}

// NewHTTPCentralUIReachabilityChecker creates a new HTTP-based reachability checker
func NewHTTPCentralUIReachabilityChecker() *HTTPCentralUIReachabilityChecker {
	return &HTTPCentralUIReachabilityChecker{
		httpClient: &http.Client{
			Timeout: httpCheckTimeout,
		},
	}
}

// IsCentralUIHostReachable performs an HTTP check to verify if the Central UI host is reachable
func (c *HTTPCentralUIReachabilityChecker) IsCentralUIHostReachable(ctx context.Context, uiHost string) (bool, error) {
	if uiHost == "" {
		return false, errors.New("UI host is empty")
	}

	// Construct the URL with https scheme
	url := fmt.Sprintf("https://%s", uiHost)

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return false, errors.Wrapf(err, "creating HTTP request for %s", url)
	}

	// Perform the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, errors.Wrapf(err, "HTTP request failed for %s", url)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Accept any response status code in the 2xx or 3xx range as reachable
	// This allows for redirects and successful responses
	return resp.StatusCode >= 200 && resp.StatusCode < 400, nil
}