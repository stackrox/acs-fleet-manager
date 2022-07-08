# ACS Fleet Manager

This repository started as a fork of the Fleet Manager Golang Template. Its original README is preserved in its own section below.

TODO: Clean up and make this ACS Fleet Manager specific.

## Quickstart

### Contributing

- Develop on top of `main` branch
- Add `// TODO(create-ticket): some explanation` near your work, so we can come back and refine it
- Merge your PRs to `main` branch

Context: We want to fix the e2e flow in parallel across many engineers quickly, but don't want to push untested, potentially minimal / simplified code to `release`. So we will develop in `main` and later on clean things up and bring them into `release`.

### Rough map

Here are the key directories to know about:

- docs/ Documentation
- internal/ ACS Fleet Management specific logic
- openapi/ Public, private (admin), and fleet synchronizer APIs
- pkg/ Non-ACS specific Fleet Management logic.
  - Examples include authentication, error handling, database connections, and more
- templates/
  - These are actually OpenShift templates for deploying jobs to OpenShift clusters

### Commands

```bash
# Install the prereqs:
# Golang 1.17+
# Docker
# ocm cli: https://github.com/openshift-online/ocm-cli  (brew/dnf)
# Node.js v12.20+  (brew/dnf)

make binary

# Generate the necessary secret files (empty placeholders)
make secrets/touch

# make db/teardown # If necessary, tear down the existing db:
make db/setup && make db/migrate
make db/login
# Postgresql commands:
  \dt                        # List tables in postgresql database
  select * from migrations;  # Run a query to view the migrations
  quit
  
# By default web (no TLS) at localhost:8000, metrics at localhost:8080, healthcheck (no TLS) at localhost:8083
./fleet-manager serve

# Debugging:
# I0308 13:36:58.977437   29044 leader_election_mgr.go:115] failed to acquire leader lease: failed to retrieve leader leases: failed to connect to `host=localhost user=fleet_manager database=serviceapitests`: dial error (dial tcp 127.0.0.1:5432: connect: connection refused); failed to connect to `host=localhost user=fleet_manager database=serviceapitests`: dial error (dial tcp 127.0.0.1:5432: connect: connection refused)
# => Check that the fleet-manager-db docker image is running
# docker ps --all
# docker restart fleet-manager-db
```

```bash
# Run some commands against the API:
# See ./docs/populating-configuration.md#interacting-with-the-fleet-manager-api
# TL;DR: Sign in to https://cloud.redhat.com, get token at https://console.redhat.com/openshift/token, login:
ocm login --token <ocm-offline-token>
# Generate a new OCM token (will expire, unlike the ocm-offline-token):
OCM_TOKEN=$(ocm token)
# Use the token in an API request, for example:
curl -H "Authorization: Bearer ${OCM_TOKEN}" http://127.0.0.1:/8000/api/dinosaurs_mgmt
```

```bash
# Setting up a local CRC cluster:
crc setup  # Takes some time to uncompress (12 GiB?!)
# Increase CRC resources (4 CPU and 9 GiB RAM seems to be too little, never comes up)
crc config set cpus 10
crc config set memory 18432
crc start  # Requires a pull secret from https://cloud.redhat.com/openshift/create/local
crc console --credentials  # (Optional) Get your login credentials, use them to login, e.g.:
# CRC includes a cached OpenShift `oc` client binary, this will set up the environment to use the cached `oc` binary:
eval $(crc oc-env)
# Login as a developer to test:
oc login -u developer -p developer https://api.crc.testing:6443
```

```bash
# OpenShift clusters have the Operator Lifecycle Manager installed by default.
# If running with a non-OpenShift Kubernetes cluster, you'll need to install the
# OLM yourself for the ACS Operator installation to work.
# Instructions: https://sdk.operatorframework.io/docs/installation/
# TL;DR:
brew install operator-sdk   # Install the operator SDK
operator-sdk olm install    # Install the OLM operator to your cluster
kubectl -n olm get pods -w  # Verify installation of OLM
```

# Fleet Manager Golang Template

This project is an example fleet management service. Fleet managers govern service 
instances across a range of cloud provider infrastructure and regions. They are 
responsible for service placement, service lifecycle including blast radius aware 
upgrades,control of the operators handling each service instance, DNS management, 
infrastructure scaling and pre-flight checks such as quota entitlement, export control, 
terms acceptance and authorization. They also provide the public APIs of our platform 
for provisioning and managing service instances.


To help you while reading the code the example service implements a simple collection
of _dinosaurs_ and their provisioning, so you can immediately know when something is 
infrastructure or business logic. Anything that talks about dinosaurs is business logic, 
which you will want to replace with your our concepts. The rest is infrastructure, and you
will probably want to preserve without change.

For a real service written using the same fleet management pattern see the
[kas-fleet-manager](https://github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager).

To contact the people that created this template go to [zulip](https://bf2.zulipchat.com/).

## Prerequisites
* [Golang 1.17+](https://golang.org/dl/)
* [Docker](https://docs.docker.com/get-docker/) - to create database
* [ocm cli](https://github.com/openshift-online/ocm-cli/releases) - ocm command line tool
* [Node.js v12.20+](https://nodejs.org/en/download/) and [npm](https://docs.npmjs.com/downloading-and-installing-node-js-and-npm)

## Using the template for the first time
The [implementation](./docs/implementation.md) talks about the main components of this template. 
To bootstrap your application, after cloning the repository. 

1. Replace _dinosaurs_ placeholder with your own business entity / objects
1. Implement code that have TODO comments
   ```go
   // TODO
   ```

## Running Fleet Manager for the first time in your local environment
Please make sure you have followed all of the prerequisites above first.  

1. Follow the [populating configuration guide](docs/populating-configuration.md)
   to prepare Fleet Manager with its needed configurations

1. Compile the Fleet Manager binary
```
make binary
```
1. Create and setup the Fleet Manager database
    - Create and setup the database container and the initial database schemas
    ```
    make db/setup && make db/migrate
    ```
    - Optional - Verify tables and records are created
    ```
    # Login to the database to get a SQL prompt
    make db/login
    ```
    ```
    # List all the tables
    serviceapitests# \dt
    ```
    ```
    # Verify that the `migrations` table contains multiple records
    serviceapitests# select * from migrations;
    ```

1. Start the Fleet Manager service in your local environment
    ```
    ./fleet-manager serve
    ```

    This will start the Fleet Manager server and it will expose its API on
    port 8000 by default

    >**NOTE**: The service has numerous feature flags which can be used to enable/disable certain features 
    of the service. Please see the [feature flag](./docs/feature-flags.md) documentation for more information.
1. Verify the local service is working
    ```
    curl -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/dinosaurs_mgmt/v1/dinosaurs
   {"kind":"DinosaurRequestList","page":1,"size":0,"total":0,"items":[]}
    ```
   >NOTE: Change _dinosaur_ to your own rest resource

   >NOTE: Make sure you are logged in to OCM through the CLI before running
          this command. Details on that can be found [here](./docs/populating-configuration.md#interacting-with-the-fleet-manager-api)

## Using the Fleet Manager service

### Interacting with Fleet Manager's API

See the [Interacting with the Fleet Manager API](docs/populating-configuration.md#interacting-with-the-fleet-manager-api)
subsection in the [Populating Configuration](docs/populating-configuration.md)
documentation

### Viewing the API docs

```
# Start Swagger UI container
make run/docs

# Launch Swagger UI and Verify from a browser: http://localhost:8082

# Remove Swagger UI conainer
make run/docs/teardown
```

### Running additional CLI commands

In addition to starting and running a Fleet Manager server, the Fleet Manager
binary provides additional commands to interact with the service (i.e. cluster
creation/scaling, Dinosaur creation, Errors list, etc.) without having to use a
REST API client.

To use these commands, run `make binary` to create the `./fleet-manager` binary.

Then run `./fleet-manager -h` for information on the additional available
commands.

### Fleet Manager Environments

The service can be run in a number of different environments. Environments are
essentially bespoke sets of configuration that the service uses to make it
function differently. Environments can be set using the `OCM_ENV` environment
variable. Below are the list of known environments and their
details.

- `development` - The `staging` OCM environment is used. Sentry is disabled.
   Debugging utilities are enabled. This should be used in local development.
   This is the default environment used when directly running the Fleet
   Manager binary and the `OCM_ENV` variable has not been set.
- `testing` - The OCM API is mocked/stubbed out, meaning network calls to OCM
   will fail. The auth service is mocked. This should be used for unit testing.
- `integration` - Identical to `testing` but using an emulated OCM API server
   to respond to OCM API calls, instead of a basic mock. This can be used for
   integration testing to mock OCM behaviour.
- `production` - Debugging utilities are disabled, Sentry is enabled.
   environment can be ignored in most development and is only used when
   the service is deployed.

The `OCM_ENV` environment variable should be set before running any Fleet
Manager binary command or Makefile target

### Running the fleet manager with an OSD cluster form infractl

Write a Cloud provider configuration file that matches the cloud provider and region used for the cluster, see `dev/config/provider-configuration-infractl-osd.yaml` for an example OSD cluster running in GCP. See the cluster creation logs in https://infra.rox.systems/cluster/YOUR_CLUSTER to locate the provider and region. See `internal/dinosaur/pkg/services/cloud_providers.go` for the provider constant. 

Enable a cluster configuration file for the OSD cluster, see `dev/config/dataplane-cluster-configuration-infractl-osd.yaml` for an example OSD cluster running in GCP. Again, see the cluster creation logs for possibly missing required fields. 

Download the kubeconfig for the cluster. Without this the fleet manager will refuse to use the cluster.

```bash
CLUSTER=... # your cluster's name
infractl artifacts $CLUSTER --download-dir ~/infra/$CLUSTER
```

Launch the fleet manager using those configuration files:

```bash
make binary && ./fleet-manager serve \
   --dataplane-cluster-config-file=$(pwd)/dev/config/dataplane-cluster-configuration-infractl-osd.yaml \
   --providers-config-file=$(pwd)/dev/config/provider-configuration-infractl-osd.yaml \
   --kubeconfig=${HOME}/infra/${CLUSTER}/kubeconfig \
   2>&1 | tee fleet-manager-serve.log
```

### Running containerized fleet-manager and fleetshard-sync

The makefile target `image/build` builds a combined image, containing both applications, `fleet-manager` and `fleetshard-sync`.

So far only `fleet-manager` can be successfully spawned from this image, because `fleetshard-sync` tries to reach `fleet-manager` at `127.0.0.1` (hard-coded).

Using e.g. the Docker CLI, `fleet-manager` can be spawned as follows:

```
docker run -it --rm -p 8000:8000 \
   -v "$(git rev-parse --show-toplevel)/config":/config \
   -v "$(git rev-parse --show-toplevel)/secrets":/secrets \
   <IMAGE REFERENCE> \
   --db-host-file secrets/db.host.internal-docker \
   --api-server-bindaddress 0.0.0.0:8000
```

Using the above command the `fleet-manager` application tries to access its database running on the host system and its API server is
reachable at `localhost` (host system).

In principle `fleetshard-sync` will be able to spawned using a command similar to the following:

```
OCM_TOKEN=$(ocm token --refresh)
docker run -it -e OCM_TOKEN --rm -p 8000:8000 \
   --entrypoint /usr/local/bin/fleetshard-sync \
   -v "$(git rev-parse --show-toplevel)/config":/config \
   -v "$(git rev-parse --show-toplevel)/secrets":/secrets \
   <IMAGE REFERENCE>
```

For this to work `fleetshard-sync` has to be modified so that `fleet-manager`'s endpoint is configurable and both containers have to be
running using a shared network so that they can access each other (TODO).

## Additional documentation
- [Adding new endpoint](docs/adding-a-new-endpoint.md)
- [Adding new CLI flag](docs/adding-new-flags.md)
- [Automated testing](docs/automated-testing.md)
- [Deploying fleet manager via Service Delivery](docs/onboarding-with-service-delivery.md)
- [Requesting credentials and accounts](docs/getting-credentials-and-accounts.md)
- [Data Plane Setup](docs/data-plane-osd-cluster-options.md)
- [Access Control](docs/access-control.md)
- [Quota Management](docs/quota-management-list-configuration.md)
- [Running the Service on an OpenShift cluster](./docs/deploying-fleet-manager-to-openshift.md)
- [Explanation of JWT token claims used across the fleet-manager](docs/jwt-claims.md)

## Contributing
See the [contributing guide](CONTRIBUTING.md) for general guidelines on how to
contribute back to the template.
