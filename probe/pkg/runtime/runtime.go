package runtime

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/probe"
	"k8s.io/apimachinery/pkg/util/wait"
)

var backoff = wait.Backoff{
	Duration: 1 * time.Second,
	Factor:   1.5,
	Jitter:   0.1,
	Steps:    15,
	Cap:      10 * time.Minute,
}

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

// Start a continuous loop of probe runs.
func (r *Runtime) Start(ctx context.Context) error {
	ticker := time.NewTicker(r.Config.RuntimeRunWaitPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "probe context cancelled")
		case <-ticker.C:
			err := r.RunSingle(ctx)
			if errors.Is(err, probe.ErrNotRecoverable) {
				return errors.Wrap(err, "probe run failed")
			}
		}
	}
}

// RunSingle executes a single probe run.
func (r *Runtime) RunSingle(ctx context.Context) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, r.Config.RuntimeRunTimeout)
	defer cancel()
	defer r.CleanUp()

	err := probe.Execute(ctxTimeout)
	if ctxErr := ctxTimeout.Err(); ctxErr != nil {
		ctxErr = errors.Wrap(ctxErr, "probe context cancelled")
		glog.Warning(ctxErr)
		return ctxErr
	}
	if err != nil {
		return errors.Wrap(err, "probe run failed")
	}
	return nil
}

// CleanUp remaining probe resources.
func (r *Runtime) CleanUp() error {
	glog.Info("probe resources have been cleaned up")
	return nil
}
