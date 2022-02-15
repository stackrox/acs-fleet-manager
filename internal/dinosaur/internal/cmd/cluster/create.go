package cluster

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/flags"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

// NewCreateCommand creates a new command for creating clusters.
func NewCreateCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new cluster",
		Long:  "Create a new cluster.",
		Run: func(cmd *cobra.Command, args []string) {
			runCreate(env, cmd, args)
		},
	}

	cmd.Flags().String(FlagRegion, "us-east-1", "Cluster region ID")
	cmd.Flags().String(FlagProvider, "aws", "Cluster provider")
	cmd.Flags().Bool(FlagMultiAZ, true, "Whether Cluster request should be Multi AZ or not")
	cmd.Flags().String(FlagProviderType, "ocm", "The provider type")

	return cmd
}

func runCreate(env *environments.Env, cmd *cobra.Command, _ []string) {
	region := flags.MustGetDefinedString(FlagRegion, cmd.Flags())
	provider := flags.MustGetDefinedString(FlagProvider, cmd.Flags())
	multiAZ := flags.MustGetBool(FlagMultiAZ, cmd.Flags())
	providerType := flags.MustGetDefinedString(FlagProviderType, cmd.Flags())

	var clusterService services.ClusterService
	env.MustResolveAll(&clusterService)

	clusterRequest := api.Cluster{
		CloudProvider: provider,
		Region:        region,
		MultiAZ:       multiAZ,
		Status:        api.ClusterAccepted,
		ProviderType:  api.ClusterProviderType(providerType),
	}

	if err := clusterService.RegisterClusterJob(&clusterRequest); err != nil {
		glog.Fatalf("Unable to create cluster request: %s", err.Error())
	}

	glog.Infoln("Cluster request has been registered and will be reconciled shortly.")
}
