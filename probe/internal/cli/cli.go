package cli

import (
	"context"
	"os"
	"os/signal"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/runtime"
)

var (
	// errInterruptSignal corresponds to a received SIGINT signal.
	errInterruptSignal = errors.New("received interrupt signal")
)

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

			ctx := context.Background()
			runResult := make(chan error, 1)

			runFunc := func() {
				if err := runtime.Start(runResult); err != nil {
					glog.Fatal(err)
				}
			}
			return handleErrors(ctx, runResult, runFunc)
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

			runFunc := func() {
				select {
				case runResult <- runtime.RunSingle(ctxTimeout):
				case <-ctxTimeout.Done():
				}
			}
			return handleErrors(ctxTimeout, runResult, runFunc)
		},
	}
	return c
}

func handleErrors(ctx context.Context, runResult chan error, runFunc func()) error {
	go runFunc()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	select {
	case <-sigs:
		glog.Error("Received SIGINT signal, shutting down ...")
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
