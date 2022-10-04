# ACS Fleet Manager
[![Dinosaur counter](https://dinosaurs.rhacs-dev.com/)](https://sourcegraph.com/search?q=context:global+repo:stackrox/acs-fleet-manager+dinosaur+count:all&patternType=standard)


ACS fleet-manager repository for the ACS managed service.


## Quickstart

### Overview

```
├── bin                 -- binary output directory  
├── cmd                 -- cmd entry points
├── config              -- various fleet-manager configurations
├── dashboards          -- grafana dashboards
├── docs                -- documentation
├── docker              -- docker images
├── dp-terraform        -- terraforming scripts for data-plane clusters
├── e2e                 -- e2e tests
├── fleetshard          -- source code for fleetshard-synchronizer
├── internal            -- internal source code
├── openapi             -- openapi specification
├── pkg                 -- pkg code
├── scripts             -- development and test scripts
├── secrets             -- secrets which are mounted to the fleet-manager
├── templates           -- fleet-manager openshift deployment templates
└── test                -- test mock servers
```

### Getting started

#### Prerequisites

* [Golang 1.17+](https://golang.org/dl/)
* [Docker](https://docs.docker.com/get-docker/) - to create database
* [ocm cli](https://github.com/openshift-online/ocm-cli/releases) - ocm command line tool
* [Node.js v12.20+](https://nodejs.org/en/download/) and [npm](https://docs.npmjs.com/downloading-and-installing-node-js-and-npm)
- A running kubernetes cluster
- Setting up configurations described [here](./docs/development/populating-configuration.md#interacting-with-the-fleet-manager-api)

#### Getting started

To run fleet-manager in different ways (i.e. with docker) please refer to [running-fleet-manager.md](./docs/development/running-fleet-manager.md).

```bash
# Export the kubeconfig path the central instance should be deployed to
$ export KUBECONFIG=/your/kubeconfig

# Sets up database, starts fleet-manager
$ make setup-dev-env

# Start fleetshard-sync
$ OCM_TOKEN=$(ocm token --refresh) CLUSTER_ID=1234567890abcdef1234567890abcdef ./fleetshard-sync

# To create a central instance
$ ./scripts/create-central.sh

# To interact with the API use
$ ./scripts/fmcurl
```

##### Prepare a cluster for the local environment
You can run ACS Fleet Manager on a local k8s cluster (colima, rancher, minikube).
This requires some preparations on a cluster:

1. Install ACS Operator
   The process is described in [operator documentation](https://github.com/stackrox/stackrox/tree/master/operator)
2. Deploy OpenShift Router
   ```
   make deploy/openshift-router
   ```
   Use `make undeploy/openshift-router` to undeploy the OpenShift router from a cluster.
   Refer to the router [repo](https://github.com/openshift/router) for more information.
3. For accessing the hostnames exposed by routes locally read the documentation [here](docs/development/test-locally-route-hosts.md)


#### Common make targets

```
# Install git-hooks, for more information see git-hooks.md [1]
$ make setup/git/hooks

# To generate code and compile binaries run
$ make all

# To only compile fleet-manager and fleetshard-synchronizer run
$ make binary

# Run API docs server
$ make run/docs

# Generate code such as openapi
$ make generate

# Testing related targets
$ make test
$ make test/e2e
$ make test/integration

# Fleet-manager database related make targets
$ make db/teardown
$ make db/setup
$ make db/migrate
```

[1] [git-hooks.md](./docs/development/git-hooks.md)

#### Background

This project was started from a fleet-manager template with an example "Dinosaur" application as a managed service.
Implementations which reference "Dinosaur" are replaced iteratively.

For a real service written using the same fleet management pattern see the
[kas-fleet-manager](https://github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager).
Original [fleet-manager template](https://github.com/bf2fc6cc711aee1a0c2a/ffm-fleet-manager-go-template).

To contact the people that created this template go to [zulip](https://bf2.zulipchat.com/).

## Additional documentation

- [Adding new endpoint](docs/development/adding-a-new-endpoint.md)
- [Deploying fleet manager via Service Delivery](docs/legacy/onboarding-with-service-delivery.md)
- [Data Plane Setup](docs/legacy/data-plane-osd-cluster-options.md)
- [Access Control](docs/legacy/access-control.md)
- [Quota Management](docs/legacy/quota-management-list-configuration.md)
- [Explanation of JWT token claims used across the fleet-manager](docs/auth/jwt-claims.md)

## Contributing

See the [contributing guide](CONTRIBUTING.md) for general guidelines on how to
contribute back to the template.
