package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/gitops"
)

func newInstallCommand() *cobra.Command {
	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Installs the gitops operator.",
		Long:  "A command that installs ArgoCD aka openshift-gitops-operator",
		RunE: func(cmd *cobra.Command, args []string) error {
			timeoutDuration, err := cmd.Flags().GetDuration("timeout")
			if err != nil {
				return fmt.Errorf("failed to parse timeout flag: %w", err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
			defer cancel()
			clusterName, err := cmd.Flags().GetString("cluster-name")
			if err != nil {
				return fmt.Errorf("failed to parse 'cluster-name' flag: %w", err)
			}
			bootstrapRev, err := cmd.Flags().GetString("bootstrap-revision")
			if err != nil {
				return fmt.Errorf("failed to parse 'bootstrap-revision' flag: %w", err)
			}
			return gitops.InstallGitopsOperator(ctx,
				gitops.WithClusterName(clusterName),
				gitops.WithBootstrapAppTargetRevision(bootstrapRev))
		},
	}
	installCmd.Flags().DurationP("timeout", "t", 5*time.Minute, "Timeout for the install operation (e.g., 30s, 1m, 2h30m), defaults to 5 minutes.")
	installCmd.Flags().String("cluster-name", "", "Optional cluster name. If not specified, the cluster name will attempt to resolve automatically.")
	installCmd.Flags().String("bootstrap-revision", "HEAD", "Bootstrap app target revision.")
	return installCmd
}
