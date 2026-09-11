package reconciler

import (
	"context"
)

// MockCentralUIReachabilityChecker is a mock implementation for testing
type MockCentralUIReachabilityChecker struct {
	reachable bool
	err       error
}

// NewMockCentralUIReachabilityChecker creates a new mock checker
func NewMockCentralUIReachabilityChecker(reachable bool, err error) *MockCentralUIReachabilityChecker {
	return &MockCentralUIReachabilityChecker{
		reachable: reachable,
		err:       err,
	}
}

// IsCentralUIHostReachable returns the mocked reachability status
func (m *MockCentralUIReachabilityChecker) IsCentralUIHostReachable(_ context.Context, _ string) (bool, error) {
	return m.reachable, m.err
}

// SetReachable sets the reachability status for the mock
func (m *MockCentralUIReachabilityChecker) SetReachable(reachable bool) {
	m.reachable = reachable
}

// SetError sets the error for the mock
func (m *MockCentralUIReachabilityChecker) SetError(err error) {
	m.err = err
}