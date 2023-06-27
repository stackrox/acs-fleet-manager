package integration

import (
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/stackrox/acs-fleet-manager/test/mocks"
)

func mockedClusterWithMetricsInfo(computeNodes int) (*clustersmgmtv1.Cluster, error) {
	clusterBuilder := mocks.GetMockClusterBuilder(nil)
	clusterNodeBuilder := clustersmgmtv1.NewClusterNodes()
	clusterNodeBuilder.Compute(computeNodes)
	clusterBuilder.Nodes(clusterNodeBuilder)
	return clusterBuilder.Build()
}
