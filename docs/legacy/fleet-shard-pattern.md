This document contains useful links on the fleetshard pattern.

# Why the fleetshard model?

See [ADR-4:Fleet manager - fleet shard local autonomy](https://architecture.appservices.tech/ap/4/)

# I need a fleet manager deployed to get initial feedback but I do not have fleetshard components, what do I do?

You could still have the first version of the fleet manager that directly pushes (via SyncSet) the CRs to the data plane clusters to deploy your service. For that you could rely on the [create syncset](https://github.com/bf2fc6cc711aee1a0c2a/ffm-fleet-manager-go-template/blob/main/pkg/client/ocm/client.go#L33) and [delete syncset]() methods in the [provisioning reconcilers](../internal/central/internal/workers/dinosaur_mgrs/provisioning_dinosaurs_mgr.go). The syncset can be updated (when some resources changes) via [update syncset](https://github.com/bf2fc6cc711aee1a0c2a/ffm-fleet-manager-go-template/blob/main/pkg/client/ocm/client.go#L34) or deleted (when a managed instance is to be deprovisioned) via [delete syncset](https://github.com/bf2fc6cc711aee1a0c2a/ffm-fleet-manager-go-template/blob/main/pkg/client/ocm/client.go#L36) methods. From this model you could iterate by building the fleetshard components and integrate them on the next phase.

# What's a ManagedCR?

See [ADR-1: ManagedCR](https://architecture.appservices.tech/ap/1/)

# How do to deal with version hand over?

See [ADR-5:Instance operator responsibility hand over](https://architecture.appservices.tech/ap/5/)

# How do I version Fleet shard operator?
See [ADR-6:Fleet shard operator - instance operator versioning](https://architecture.appservices.tech/ap/6/)

# How to handle IngressController?

See [ADR-56: IngressController per cloud Availability Zone](https://architecture.appservices.tech/adr/56/)

# Who's responsible for the creation of DNS records?

See [ADR-57: Fleetshard tells fleet manager what DNS records to create for an instance](https://architecture.appservices.tech/adr/57/)

# How do I create an AddOn of my Operator?

To create an AddOn of your operator, please have a look at [AddOn Flow documentation](https://gitlab.cee.redhat.com/service/managed-tenants#add-ons-flow-metadata-repository)

# How it is done by Others?
The following documents highlights how this pattern is done by the Kafka service

- [ADR-37: Managed Kafka Fleetshard Operator](https://architecture.appservices.tech/adr/37/)
- [ADR-38: Managed Kafka Fleetshard Synchronizer](https://architecture.appservices.tech/adr/38/)
- [ADR-40: API Contract betwwen Kafka Fleet Manager and Fleetshard Synchronizer](https://architecture.appservices.tech/adr/40/). A base of that API Contract is available in [Agent OpenAPI Specification](../openapi/fleet-manager-private.yaml). See the [customizing api documentation](./customizing-openapi-spec.md#private-cluster-agent-endpoints) on how to modify it.
- ManagedKafka Fleetshard AddOn
    - [QE](https://gitlab.cee.redhat.com/service/managed-tenants/-/tree/main/addons/kas-fleetshard-operator-qe/metadata) used for Staging workload
    - [Prod](https://gitlab.cee.redhat.com/service/managed-tenants/-/tree/main/addons/kas-fleetshard-operator/metadata/production) used for Production workload
- Managed Kafka Addon:
    - [QE](https://gitlab.cee.redhat.com/service/managed-tenants/-/tree/main/addons/rhosak-qe/metadata) used for Staging workload
    - [Prod](https://gitlab.cee.redhat.com/service/managed-tenants/-/tree/main/addons/rhosak/metadata/production) used for Production workload
