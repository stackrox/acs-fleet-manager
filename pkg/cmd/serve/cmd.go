package serve

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/environments"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

func NewServeCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve the fleet-manager",
		Long:  "Serve the Dinosaur Service Fleet Manager.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			err := env.CreateServices()
			if err != nil {
				glog.Fatalf("Unable to initialize environment: %s", err.Error())
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Cancel the context when we get a signal...
			ch := make(chan os.Signal, 1)
			signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				select {
				case <-ch:
					cancel()
				case <-ctx.Done():
				}
			}()

			env.Run(ctx)
		},
	}
	return cmd
}
