package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/stackrox/acs-fleet-manager/probe/config"
)

// Runtime performs a probe run against fleet manager.
type Runtime struct {
	config *config.Config
}

// New creates a new runtime.
func New(config *config.Config) (*Runtime, error) {
	return &Runtime{
		config: config,
	}, nil
}

// RunSingle executes a single probe run.
func (r *Runtime) RunSingle(ctx context.Context) error {
	// Dummy run
	fmt.Println("start run")
	time.Sleep(5 * time.Second)
	fmt.Println("end run")
	return nil
}

// Stop the probe run.
func (r *Runtime) Stop() error {
	return nil
}
