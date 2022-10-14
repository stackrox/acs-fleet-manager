# ACS MS Test Environment

The `dev/env` directory contains scripts for bringing up a complete ACS MS test environment on different
types of cluster (currently: Minikube, Colima, Infra OpenShift, OpenShift CI). The following
components are set up:

* A Postgres database
* Fleet Manager
* Fleetshard Sync
* RHACS Operator

The RHACS operator can be installed from OpenShift marketplace or Quay. Images for Fleet Manager & Fleetshard Sync can either be pulled from Quay or built directly from the source.

The following scripts exist currently in `dev/env/scripts`:

* `lib.sh`: Basic initialization and library script for the other executable scripts.
* `apply` & `delete`: Convenience scripts for applying and deleting Kubernetes resources supporting environment interpolation.
* `port-forwarding`: Convenient abstraction layer for kubectl port-forwarding.
* `bootstrap.sh`: Sets up the basic environment: creates namespaces, injects image-pull-secrets if necessary, installs OLM (if required), installs RHACS operator (if desired), pulls required images, etc.
* `up.sh`: Brings up the ACS MS environment consisting of the database, `fleet-manager` and `fleetshard-sync`.
* `down.sh`: Deletes the resources created by `up.sh`.

The scripts can be configured using environment variables, the most important options being:

* `CLUSTER_TYPE`: Can be `minikube`, `colima`, `openshift-ci`, `infra-openshift`). Will be
  auto-sensed in most situations depending on the cluster name.
* `FLEET_MANAGER_IMAGE`: Reference for an `acs-fleet-manager` image. If unset, build a fresh image from the current source and deploy that.
* `AUTH_TYPE`: Can be `OCM` (in which case a new token will be created automatically using `ocm token --refresh`) or `STATIC_TOKEN`, in which case a valid static token is expected in the environment variable `STATIC_TOKEN`.
* `STATIC_TOKEN_ADMIN`: Needs to contain a valid admin token (can be found in BitWarden)
* `QUAY_USER` & `QUAY_TOKEN`: Mandatory setting in case images need to be pulled from Quay.

## Prerequisites

Currently supported cluster types are:
* Local Minikube
* Remote Infra OpenShift 4.x
* OpenShift CI

Required tools:
* standard Unix environment with Bash
* `docker` CLI (or replacement)
* Minikube (if deploying to Minikube)
* operator-sdk (if deploying to clusters not having access to OpenShift Marketplace, like Minikube)
* `yq` & `jq`
* `kubectl` or `oc`

## Running with DNS tests

The test suite has auto-sensing logic built in to skip DNS e2e tests when the test environment does  not support execution of DNS e2e tests. Currently this is only supported in OpenShift environments.

To run the DNS e2e tests additionally to the default e2e test setup the cluster you're running against needs to have the openshift Route Custom Resource Definition installed and you need to set following environment variables:

```
export ROUTE53_ACCESS_KEY="<key-id>"
export ROUTE53_SECRET_ACCESS_KEY="<secret-key>"

# Depending on cluster type and its default configuration you might need
export ENABLE_CENTRAL_EXTERNAL_CERTIFICATE_DEFAULT=true

# If the domain you test against is not the default dev domain
export CENTRAL_DOMAIN_NAME="<domain>"
```

## Examples

### Minikube

Make sure that Minikube is running with options such as:
```
$ minikube start --memory=6G \
                 --cpus=2 \
                 --apiserver-port=8443 \
                 --embed-certs=true \
                 --delete-on-failure=true \
                 --driver=hyperkit # For example
```

and that the `docker` CLI is in `PATH` (if not, export `DOCKER=...` accordingly). Furthermore, prepare your environment by setting:
* `QUAY_USER`
* `QUAY_TOKEN`
* `STATIC_TOKEN`
* `STATIC_TOKEN_ADMIN`

The primary way for executing the e2e test suite is by calling
```
$ ./.openshift-ci/test/e2e.sh
```

In certain situations it is also useful to be able to execute the respective building blocks manually:

```
$ dev/env/scripts/bootstrap.sh # For bootstrapping the basic environment
$ dev/env/scripts/up.sh        # For brining up the Managed Services components
```

Then, after fleet-manager's leader election is complete (check it's logs), you can run the e2e test
suite manually:
```
make test/e2e
```

### Colima

Make sure that Colima is running with options such as:
```
$ colima start -c 4 -d 60 -m 16 -k
```

and that the `colima` CLI is in `PATH` (if not, export `DOCKER=/path/to/bin/colima nerdctl -- -n k8s.io` accordingly). Furthermore, prepare your environment by setting:
* `QUAY_USER`
* `QUAY_TOKEN`
* `STATIC_TOKEN`
* `STATIC_TOKEN_ADMIN`

### CRC

CRC needs a lot of resources and so does a Central tenant. At least the following resource settings were required to make the test succeed on CRC.

```
crc config set memory 18432
crc config set cpus 7
```

There's currently no automated way to upload the fleet-manager image to CRC. Set the `FLEET_MANAGER_IMAGE` environment variable to an available Image in quay or build locally and load it into CRC registry manually.

Furthermore, prepare your environment by setting:
* `QUAY_USER`
* `QUAY_TOKEN`
* `STATIC_TOKEN`
* `STATIC_TOKEN_ADMIN`
