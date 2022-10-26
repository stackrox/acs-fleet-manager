package probe

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
)

var (
	// ErrNotRecoverable should lead to a graceful service shutdown.
	ErrNotRecoverable = errors.New("not recoverable error")
)

// Execute the probe of the fleet manager API.
func Execute(ctx context.Context) error {
	// Dummy run
	glog.Info("probe run has been started")
	concurrency.WaitWithTimeout(ctx, 5*time.Second)
	glog.Info("probe run has ended")
	return nil
}
