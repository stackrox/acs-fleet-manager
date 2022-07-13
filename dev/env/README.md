# ACS MS Test Environment

This directory contains scripts for bringing up a complete ACS MS test environment on different types of cluster (currently: Minikube, Infra OpenShift, OpenShift CI). The following components are set up:

* A Postgres database
* Fleet Manager
* Fleetshard Sync
* RHACS Operator

The RHACS operator can be installed from OpenShift marketplace or Quay. Images for Fleet Manager & Fleetshard Sync can either be pulled from Quay or built directly from the source.

The following scripts exist currently:

* `lib.sh`: Basic initialization and library script for the other executable scripts.
* `apply` & `delete`: Convenience scripts for applying and deleting Kubernetes resources supporting environment interpolation.
* `port-forwarding`: Convenient abstraction layer for kubectl port-forwarding.
* `bootstrap.sh`: Sets up the basic environment: creates namespaces, injects image-pull-secrets if necessary, installs OLM (if required), isntalls RHACS operator (if desired), pulls required images, etc.
* `up.sh`: Brings up the ACS MS environment consisting of the database, `fleet-manager` and `fleetshard-sync`.
* `down.sh`: Deletes the resources created by `up.sh`.

The scripts can be configured using environment variables, the most important options being:

* `CLUSTER_TYPE`: Can be `minikube`, `openshift-ci`, `infra-openshift`).
* `FLEET_MANAGER_IMAGE`: Reference for an `acs-fleet-manager` image. If unset, build a fresh image from the current source and deploy that.
* `AUTH_TYPE`: Can be `OCM` (in which case a new token will be created automatically using `ocm token --refresh`) or `STATIC_TOKEN`, in which case a valid static token is expected in the environment variable `STATIC_TOKEN`.
* `QUAY_USER` & `QUAY_TOKEN`: Mandatory setting in case images need to be pulled from Quay.

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
* `STATIC_TOKEN` for `AUTH_TYPE=STATIC_TOKEN` or `OCM_TOKEN` for `AUTH_TYPE=OCM`

Then do:

```
$ dev/env/scripts/bootstrap.sh
$ dev/env/scripts/up.sh
```

Then, in order to run the e2e test suite:
```
make test/e2e
```

If the goal is to run the e2e test suite, one can also execute the e2e entrypoint for OpenShift CI, which also works in different execution environments:

```
$ ./.openshift-ci/test/e2e.sh
```
