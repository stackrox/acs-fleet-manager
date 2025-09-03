package centralmgrs

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

// UIReachabilityChecker checks if a Central UI is reachable from the internet
type UIReachabilityChecker interface {
	IsReachable(ctx context.Context, uiHost string) (bool, error)
}

// HTTPUIReachabilityChecker is the default implementation that performs actual HTTP checks
type HTTPUIReachabilityChecker struct {
	httpClient *http.Client
}

// NewHTTPUIReachabilityChecker creates a new HTTP-based reachability checker
func NewHTTPUIReachabilityChecker() *HTTPUIReachabilityChecker {
	return &HTTPUIReachabilityChecker{
		httpClient: &http.Client{
			Timeout: httpCheckTimeout,
			// Don't follow redirects automatically, we want to check the exact host
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// IsReachable performs an HTTP HEAD request to verify if the Central UI host is reachable
func (c *HTTPUIReachabilityChecker) IsReachable(ctx context.Context, uiHost string) (bool, error) {
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
		// Network errors mean the host is not reachable
		return false, nil
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Accept any response status code in the 2xx or 3xx range as "reachable"
	// This indicates the DNS resolved and the server responded
	return resp.StatusCode >= 200 && resp.StatusCode < 400, nil
}

// MockUIReachabilityChecker is a mock implementation for testing
type MockUIReachabilityChecker struct {
	reachable bool
	err       error
}

// NewMockUIReachabilityChecker creates a new mock checker
func NewMockUIReachabilityChecker(reachable bool, err error) *MockUIReachabilityChecker {
	return &MockUIReachabilityChecker{
		reachable: reachable,
		err:       err,
	}
}

// IsReachable returns the mocked reachability status
func (m *MockUIReachabilityChecker) IsReachable(_ context.Context, _ string) (bool, error) {
	return m.reachable, m.err
}

// SetReachable sets the reachability status for the mock
func (m *MockUIReachabilityChecker) SetReachable(reachable bool) {
	m.reachable = reachable
}

// SetError sets the error for the mock
func (m *MockUIReachabilityChecker) SetError(err error) {
	m.err = err
}