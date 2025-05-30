# ACS Fleet Manager

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

* [Golang 1.23+](https://golang.org/dl/)
* [Docker](https://docs.docker.com/get-docker/) - to create database
* [ocm cli](https://github.com/openshift-online/ocm-cli/releases) - ocm command line tool
* [Node.js v12.20+](https://nodejs.org/en/download/) and [npm](https://docs.npmjs.com/downloading-and-installing-node-js-and-npm)
* [Java](https://openjdk.org/) - for openAPI generation
* IDE with [EditorConfig](https://editorconfig.org/) support enabled:
  - there is a [plugin for GoLand](https://www.jetbrains.com/help/go/configuring-code-style.html#editorconfig)
  - there is an [extension for VSCode](https://marketplace.visualstudio.com/items?itemName=EditorConfig.EditorConfig)
* A running kubernetes cluster

  Guide: [setup-test-environment.md](./docs/development/setup-test-environment.md#prepare-the-environment)
* Setting up configurations described [here](./docs/development/populating-configuration.md#interacting-with-the-fleet-manager-api)

#### Setup pre-commit
```shell
$ make setup/git/hooks
```

### Test local setup
```shell
# build binaries and generate openAPI
$ make all
```

#### Supported cluster types:

* Local: Minikube, **Colima** (recommended for macOS), **Kind** (recommended for linux), Rancher Desktop, CRC
* Remote: Infra OpenShift flavors*, OpenShift CI

\* For 'OpenShift 4.x' flavor there's a couple of caveats to avoid tls certificate validation errors
1. Set `trusted-certs-enabled` ("Should trusted certificates be created" in UI) parameter to `true` in order to generate trusted TLS certificates.
2. Use `oc login` instead of the downloaded `KUBECONFIG` file. The issue is tracked in [ROX-25798](https://issues.redhat.com/browse/ROX-25798).

#### Getting started

To run fleet-manager in different ways (i.e. on a test cluster) please refer to [running-fleet-manager.md](./docs/development/running-fleet-manager.md).

```bash
# Export the kubeconfig path the central instance should be deployed to
$ export KUBECONFIG=/your/kubeconfig

# Bootstrap the environment
$ make deploy/bootstrap

# Sets up database, starts fleet-manager and fleetshard-sync
$ make deploy/dev

# To create a central instance
$ ./scripts/create-central.sh

# To interact with the API use
$ ./scripts/fmcurl

# To build and re-deploy the fleet-manager and fleetshard-sync locally run:
$ make deploy/dev-fast
```

#### Common make targets

```shell
# To generate code and compile binaries run
$ make all

# To only compile fleet-manager and fleetshard-synchronizer run
$ make binary

# Run API docs server
$ make run/docs

# Generate code such as openapi
$ make generate

# Prepare dev environment for deployment
$ make deploy/bootstrap
# Deploy changes to the test cluster [2]
$ make deploy/dev

# Testing related targets
$ make test
$ make test/e2e
$ make test/integration

# Fleet-manager database related make targets
$ make db/teardown
$ make db/setup
$ make db/migrate
```

* [1] [git-hooks.md](./docs/development/git-hooks.md)
* [2] [setup-test-environment.md](./docs/development/setup-test-environment.md)

#### Background

This project was started from a fleet-manager template with an example "Dinosaur" application as a managed service.

Another fleet manager which was built from the same template [kas-fleet-manager](https://github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager).
Original [fleet-manager template](https://github.com/bf2fc6cc711aee1a0c2a/ffm-fleet-manager-go-template).

To contact the people that created this template go to [zulip](https://bf2.zulipchat.com/).

## Additional documentation

- [Adding new endpoint](docs/development/adding-a-new-endpoint.md)
- [Deploying fleet manager via Service Delivery](docs/legacy/onboarding-with-service-delivery.md)
- [Data Plane Setup](docs/legacy/data-plane-osd-cluster-options.md)
- [Access Control](docs/legacy/access-control.md)
- [Quota Management](docs/quota/quota.md)
- [Explanation of JWT token claims used across the fleet-manager](docs/auth/jwt-claims.md)
- [Gitops: configure ACS Central and ACS Operator](docs/development/gitops-configure-central-and-operator.md)

## Contributing

See the [contributing guide](CONTRIBUTING.md) for general guidelines on how to
contribute back to the template.
