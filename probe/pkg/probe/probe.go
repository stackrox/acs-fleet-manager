package probe

import (
	"context"
	"time"

	"github.com/golang/glog"
)

// Execute the probe of the fleet manager API.
func Execute(ctx context.Context) error {
	// Dummy run
	glog.Info("probe run has been started")
	time.Sleep(5 * time.Second)
	glog.Info("probe run has ended")
	return nil
}
