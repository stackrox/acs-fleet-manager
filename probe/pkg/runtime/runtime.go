package runtime

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/probe"
	"github.com/stackrox/rox/pkg/utils"
)

// Runtime performs a probe run against fleet manager.
type Runtime struct {
	Config *config.Config
}

// New creates a new runtime.
func New() (*Runtime, error) {
	config, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load configuration")
	}

	return &Runtime{
		Config: config,
	}, nil
}

// RunLoop a continuous loop of probe runs.
func (r *Runtime) RunLoop(ctx context.Context) error {
	ticker := time.NewTicker(r.Config.RuntimeRunWaitPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "probe context invalid")
		case <-ticker.C:
			if err := r.RunSingle(ctx); err != nil {
				glog.Warning(err)
			}
		}
	}
}

// RunSingle executes a single probe run.
func (r *Runtime) RunSingle(ctx context.Context) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, r.Config.RuntimeRunTimeout)
	defer cancel()
	defer r.CleanUp()

	if err := probe.Execute(ctxTimeout); err != nil {
		return errors.Wrap(err, "probe run failed")
	}
	return nil
}

// CleanUp remaining probe resources.
func (r *Runtime) CleanUp() error {
	glog.Info("probe resources have been cleaned up")
	return nil
}
