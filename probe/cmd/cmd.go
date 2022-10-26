package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/runtime"
)

// errInterruptSignal corresponds to a received SIGINT signal.
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
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, err := runtime.New()
			if err != nil {
				return errors.Wrap(err, "failed to create runtime")
			}
			defer runtime.Stop()

			go func() {
				if err := runtime.Start(); err != nil {
					glog.Fatal(err)
				}
			}()

			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, os.Interrupt)
			defer signal.Stop(sigs)

			<-sigs
			return errInterruptSignal
		},
	}
	return c
}

func runCommand() *cobra.Command {
	c := &cobra.Command{
		SilenceUsage: true,
		Use:          "run",
		Short:        "Run a single probe run.",
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, err := runtime.New()
			if err != nil {
				return errors.Wrap(err, "failed to create runtime")
			}
			defer runtime.Stop()

			ctxTimeout, cancel := context.WithTimeout(context.Background(), runtime.Config.RuntimeRunTimeout)
			defer cancel()

			runResult := make(chan error, 1)
			go func() {
				runResult <- runtime.RunSingle(ctxTimeout)
			}()

			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, os.Interrupt)
			defer signal.Stop(sigs)

			select {
			case <-sigs:
				return errInterruptSignal
			case err := <-runResult:
				return errors.Wrap(err, "probe run failed")
			case <-ctxTimeout.Done():
				return errors.Wrap(ctxTimeout.Err(), "probe run failed")
			}
		},
	}
	return c
}
