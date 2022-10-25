package cmd

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/runtime"
	"github.com/stackrox/rox/pkg/errox"
)

var errInterruptSignal error = errors.New("received interrupt signal")

// Command builds the root CLI command.
func Command() *cobra.Command {
	c := &cobra.Command{
		SilenceUsage: true,
		Use:          os.Args[0],
		Long:         "Probe is a service that verifies the availability of ACS fleet manager.",
	}
	c.AddCommand(
		startCommand(),
		runCommand(),
	)
	return c
}

func startCommand() *cobra.Command {
	c := &cobra.Command{
		SilenceUsage: true,
		Use:          "start",
		Short:        "Start a continuous loop of probe runs.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return errox.InvalidArgs.New("expected no arguments; please check usage")
			}
			config, err := config.GetConfig()
			if err != nil {
				return errors.Wrap(err, "failed to load configuration")
			}

			for {
				if err := executeProbe(config); err != nil {
					if errors.Is(err, errInterruptSignal) {
						return err
					}
					glog.Errorf("%+v", err)
				}
				time.Sleep(config.RuntimeRunWaitPeriod)
			}
		},
	}
	return c
}

func runCommand() *cobra.Command {
	c := &cobra.Command{
		SilenceUsage: true,
		Use:          "run",
		Short:        "Run a single probe run.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return errox.InvalidArgs.New("expected no arguments; please check usage")
			}
			config, err := config.GetConfig()
			if err != nil {
				return errors.Wrap(err, "failed to load configuration")
			}
			return executeProbe(config)
		},
	}
	return c
}

func executeProbe(config *config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), config.RuntimeRunTimeout)
	defer cancel()

	runtime, err := runtime.New(config)
	if err != nil {
		return errors.Wrap(err, "failed to initialize probe")
	}
	defer func() {
		runtime.Stop()
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	defer signal.Stop(sigs)

	runResult := make(chan error, 1)
	go func() {
		runResult <- runtime.RunSingle(ctx)
	}()

	select {
	case <-sigs:
		return errInterruptSignal
	case err := <-runResult:
		if err != nil {
			return errors.Wrap(err, "probe run failed")
		}
		return nil
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "probe run failed")
	}
}
