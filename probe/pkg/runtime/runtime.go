package runtime

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/probe"
	"github.com/stackrox/rox/pkg/concurrency"
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
	glog.Infof("probe service has been started")

	config, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load configuration")
	}

	return &Runtime{
		Config: config,
	}, nil
}

// Start a continuous loop of probe runs.
func (r *Runtime) Start(runResult chan error) error {
	ticker := concurrency.NewRetryTicker(func(ctx context.Context) (timeToNextTick time.Duration, err error) {
		if err := r.RunSingle(ctx); err != nil {
			if errors.Is(err, concurrency.ErrNonRecoverable) {
				runResult <- err
			}
			return 0, errors.Wrap(err, "failed to execute single probe run")
		}
		return r.Config.RuntimeRunWaitPeriod, nil
	}, r.Config.RuntimeRunTimeout, backoff)

	return errors.Wrap(ticker.Start(), "failed to start ticker")
}

// RunSingle executes a single probe run.
func (r *Runtime) RunSingle(ctx context.Context) error {
	return errors.Wrap(probe.Execute(ctx), "failed to execute the probe")
}

// Stop the probe.
func (r *Runtime) Stop() error {
	glog.Info("probe service has been stopped")
	return nil
}
