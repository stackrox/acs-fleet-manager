# Cluster Management Interface and Implementations

## Why

1. To improve the architecture of fleet-manager so that when there are changes to OCM, it should only have very small impact on the code base. At the moment, more components are directly depending on OCM than it should be, e.g. [`clusters_mgr.go`](../../internal/dinosaur/pkg/workers/clusters_mgr.go), and this makes it difficult to change the code base if OCM changes.  
2. To allow us easily open source the project. With the right interface/contract in place for cluster management, OCM will just become an implementation detail of the interface. New implementations can then be added by us or the upstream community to support different mechanisms to manage OpenShift/Kubernetes clusters.

## How

The following diagram describes the new architecture for cluster management functions:

![Cluster Management Interface](../images/cluster-management-abstraction.png)

Compared to the current implementation, the main changes are:

1. Introducing a new `Provider` interface for cluster management. This new interface presents providers of OpenShift/Kubernetes clusters, and we will have two implementations for it: one for OCM, and another for standalone clusters which are already terraformed manually outside of fleet-manager.
2. Each cluster could have different `Provider` implementation, this information needs to be stored. The `api.Cluster` type is updated to include additional fields to store this information.
3. A new `ProviderFactory` interface is added which will return the concrete `Provider` implementation based on the `Providertype` value in each `api.Cluster` instance.
4. Modify the [`ClusterService`](../../internal/dinosaur/pkg/services/clusters.go) class to depend on the new `ProviderFactory`, and remove all dependencies on the OCM client.
5. Modify the [`ClusterManager`](../../internal/dinosaur/pkg/workers/clusters_mgr.go) class to only depend on the `ClusterService`. It should not have any direct dependencies on the OCM clients.
