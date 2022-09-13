package cluster

import (
	"encoding/json"

	glog "github.com/stackrox/acs-fleet-manager/pkg/logging"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/flags"
)

// NewScaleCommand creates a new command for scaling Compute nodes in a OSD cluster
func NewScaleCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scale",
		Short: "Scale the managed services Compute nodes in a OSD cluster",
		Long:  "Scale Compute nodes (up or down) in a OSD cluster.",
	}

	cmd.AddCommand(
		NewScaleUpCommand(env),
		NewScaleDownCommand(env),
	)
	return cmd
}

// NewScaleUpCommand creates a new command for scaling up Compute nodes in a OSD cluster
func NewScaleUpCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Scale up a node",
		Long:  "Scale up Compute nodes in a OSD cluster.",
		Run: func(cmd *cobra.Command, args []string) {
			runScaleUp(env, cmd, args)
		},
	}
	cmd.Flags().String(FlagClusterID, "", "Cluster ID")
	return cmd
}

// NewScaleDownCommand creates a new command for scaling down Compute nodes in a OSD cluster
func NewScaleDownCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "down",
		Short: "Scale down a node",
		Long:  "Scale down Compute nodes in a OSD cluster.",
		Run: func(cmd *cobra.Command, args []string) {
			runScaleDown(env, cmd, args)
		},
	}
	cmd.Flags().String(FlagClusterID, "", "Cluster ID")
	return cmd
}

func runScaleUp(env *environments.Env, cmd *cobra.Command, _ []string) {
	clusterID := flags.MustGetDefinedString(FlagClusterID, cmd.Flags())
	var clusterService services.ClusterService
	env.MustResolveAll(&clusterService)

	// scale up compute nodes
	cluster, err := clusterService.ScaleUpComputeNodes(clusterID, constants.DefaultClusterNodeScaleIncrement)
	if err != nil {
		glog.Fatalf("Unable to scale up compute nodes: %s", err.Error())
	}

	// print the output
	if indentedCluster, err := json.Marshal(cluster); err != nil {
		glog.Fatalf("Unable to marshal cluster: %s", err.Error())
	} else {
		glog.V(10).Infof("%s", string(indentedCluster))
	}
}

func runScaleDown(env *environments.Env, cmd *cobra.Command, _ []string) {
	clusterID := flags.MustGetDefinedString(FlagClusterID, cmd.Flags())
	var clusterService services.ClusterService
	env.MustResolveAll(&clusterService)

	// scale down compute nodes
	cluster, err := clusterService.ScaleDownComputeNodes(clusterID, constants.DefaultClusterNodeScaleIncrement)
	if err != nil {
		glog.Fatalf("Unable to scale down compute nodes: %s", err.Error())
	}

	// print the outputs
	if indentedCluster, err := json.Marshal(cluster); err != nil {
		glog.Fatalf("Unable to marshal cluster: %s", err.Error())
	} else {
		glog.V(10).Infof("%s", string(indentedCluster))
	}
}
