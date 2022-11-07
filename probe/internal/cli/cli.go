// Package cli ...
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

// CLI defines the command line interface of the probe.
type CLI struct {
	runtime *runtime.Runtime
}

// New creates a CLI.
func New() (*CLI, error) {
	runtime, err := runtime.New()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create runtime")
	}
	return &CLI{runtime: runtime}, nil
}

// Command builds the root CLI command.
func (cli *CLI) Command() *cobra.Command {
	c := &cobra.Command{
		SilenceUsage: true,
		Use:          os.Args[0],
		Long:         "Probe is a service that verifies the availability of ACS fleet manager.",
	}
	c.AddCommand(
		cli.startCommand(),
		cli.runCommand(),
	)
	return c
}

func (cli *CLI) startCommand() *cobra.Command {
	c := &cobra.Command{
		SilenceUsage: true,
		Use:          "start",
		Short:        "Start a continuous loop of probe runs.",
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.handleInterrupt(cli.runtime.RunLoop)
		},
	}
	return c
}

func (cli *CLI) runCommand() *cobra.Command {
	c := &cobra.Command{
		SilenceUsage: true,
		Use:          "run",
		Short:        "Run a single probe run.",
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.handleInterrupt(cli.runtime.RunSingle)
		},
	}
	return c
}

func (cli *CLI) handleInterrupt(runFunc func(context.Context) error) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		<-sigs
		glog.Error("received SIGINT signal, shutting down ...")
		cancel()
	}()

	err := runFunc(ctx)
	if errors.Is(err, context.Canceled) {
		return errInterruptSignal
	}
	return err
}
