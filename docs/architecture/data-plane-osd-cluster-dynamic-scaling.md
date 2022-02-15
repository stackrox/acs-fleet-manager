# Data Plane OSD Cluster Dynamic Scaling functionality

## Description

This documents explains how Data Plane OSD Cluster Dynamic Scaling functionality
has been implemented.

Data Plane OSD Cluster Dynamic Scaling functionality is intended to deal with:
* Dynamic scaling of OSD Compute Nodes
* OSD Cluster Status capacity calculation
* Dynamic scaling of OSD Clusters

Dynamic scaling evaluation/trigger for a given Data Plane OSD cluster is
initiated and performed by Fleet Shard Operator on the data plane, calling
the `/agent-clusters/{cluster_id}/status` endpoint provided by Fleet Manager.
The information provided when this call is performed is defined in the private
OpenAPI spec file in the `DataPlaneClusterUpdateStatusRequest` data type.

In order for the dynamic scaling evaluation to be processed the following conditions
need to happen:
* The agent must be in a 'ready' state
* The OSD cluster must be in one of the states accepted by the dynamic scaling
  functionality (implementation details on code)