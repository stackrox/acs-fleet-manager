package probe

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
)

// Execute the probe of the fleet manager API.
func Execute(ctx context.Context) error {
	// Dummy run
	glog.Info("probe run has been started")
	defer glog.Info("probe run has ended")

	if expired := concurrency.WaitWithTimeout(ctx, 5*time.Second); expired {
		return errors.Wrap(ctx.Err(), "probe run expired")
	}
	return nil
}
