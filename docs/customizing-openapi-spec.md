# Customizing the Fleet Manager OpenAPI Specification
This document outlines the different OpenAPI specifications defined as part of
Fleet Manager's API.
It also shows what needs to be changed in order to customize the Fleet Manager
OpenAPI specifications to your own requirements.

  - [OpenAPI specifications overview](#openapi-specifications-overview)
  - [Generate Go code from OpenAPI specifications](#generate-go-code-from-openapi-specifications)
  - [Public endpoints](#public-endpoints)
  - [Private cluster agent endpoints](#private-cluster-agent-endpoints)
  - [Private admin endpoints](#private-admin-endpoints)

## OpenAPI specifications overview

Different OpenAPI specification files are provided as part of Fleet Manager's API:
* Public endpoints OpenAPI specification. Defined in the [openapi/fleet-manager.yaml](../openapi/fleet-manager.yaml) file
* Private cluster agent endpoints OpenAPI specification. Defined in the [openapi/fleet-manager-private.yaml](../openapi/fleet-manager-private.yaml) file
* Private Admin endpoints OpenAPI specification. Defined in the [openapi/fleet-manager-private-admin](../openapi/fleet-manager-private-admin.yaml) file

## Generate Go code from OpenAPI specifications

A set of Makefile targets are provided to generate the Go types and methods from the OpenAPI specifications
defined in the `openapi` directory

To generate the Go code from all OpenAPI specifications run:
```
make openapi/generate
```

To generate the Go code from the Fleet Manager public endpoints OpenAPI specification run:
```
make openapi/generate/public
```

To generate the Go code from the Fleet Manager private agent endpoints OpenAPI specification run:
```
make openapi/generate/private
```

To generate the Go code from the the Fleet manager private admin endpoints OpenAPI specification run:
```
make openapi/generate/admin
```

## Public endpoints

The [fleet-manager](../openapi/fleet-manager.yaml) OpenAPI spec defines the endpoints provided as part of Fleet Manager's public API

##  Private cluster agent endpoints
The [fleet-manager-private](../openapi/fleet-manager-private.yaml) OpenAPI spec defines the endpoints used by the agent for data plane communications.

### General Customization
The following are some general things that you can update or add to this template to meet your requirements.
- This template currently has references to `Dinosaur(s)` as an example service. Please change references of this to the name of your service.
- Any endpoints that should only be used by an agent or by other internal components should be added here.

Further customizations for existing endpoints are described in the following sections below.

### (GET) ​/api​/dinosaurs_mgmt​/v1​/agent-clusters​/{id}
This endpoint is intended to be used by the agent in a given data plane cluster to get the agents' own configuration from the control plane.
The configuration is defined in the `DataplaneClusterAgentConfig` OpenAPI data type

##### Customizations
Modify this OpenAPI type in the OpenAPI spec to define the information you to be part of the data plane cluster agent configuration

### (PUT) /api/dinosaurs_mgmt/v1/agent-clusters/{id}/status
This endpoint is intended to be used by the agent to report to the control plane the status of a data plane cluster.

#### Request
The **DataPlaneClusterUpdateStatusRequest** is the schema used for the request body for this endpoint. This defines multiple fields relating to the status of a cluster as well as additional information like availability of the service and service operator and their versions.

##### Customization
Modify this OpenAPI type in the OpenAPI spec to define the information you want to report from the agent, so it can be used in the control plane code

#### Response

##### Customization
The template currently has three example responses defined. The following changes can be done here:
- Addition of any other responses can be added here to suit your requirements.
- Update any references of 'Dinosaur' to the name of your service.

### (GET) /api/dinosaurs_mgmt/v1/agent-clusters/{id}/dinosaurs
This endpoint is intended to be used by the agent to get a list of existing or new custom resources
for instances/clusters (i.e. 'ManagedDinosaur') that is currently or will be managed by the agent on the data plane cluster.

#### Response
##### Customization
The template currently has three example responses defined. The following changes can be done here:
- Addition of any other responses can be added here to suit your requirements.
- Update any references of 'Dinosaur' to the name of your service.

Further customizations for specific responses can be seen in the sections below.

###### 200
**ManagedDinosaurList** is the schema used to define the response body for this endpoint. The fields for each 'Managed Dinosaur' custom resource is defined by the schema **ManagedDinosaur**.

### (PUT) /api/dinosaurs_mgmt/v1/agent-clusters/{id}/dinosaurs/status
This endpoint is intended to be used by the agent to report to the control plane the statuses of instances or clusters managed by the agent (i.e. 'Dinosaur' clusters).

#### Request
The **DataPlaneDinosaurStatusUpdateRequest** is the schema used for the request body for this endpoint. The fields in each 'Dinosaur' status is defined by the **DataPlaneDinosaurStatus** schema.

##### Customization
Modify this OpenAPI type in the OpenAPI spec to define the information you want to report from the agent, so it can be used in the control plane code

#### Response
##### Customization
The template currently has three example responses defined. The following changes can be done here:
- Addition of any other responses can be added here to suit your requirements.
- Update any references of 'Dinosaur' to the name of your service.

## Private admin endpoints

The [fleet-manager-private-admin](../openapi/fleet-manager-private-admin.yaml) OpenAPI spec defines the private admin endpoints intended to do privileged-level operations related to fleet manager, like performing upgrades of the Dinosaur instance versions
