# ACS MS Test Environment

## Overview
The `dev/env` directory contains scripts for bringing up a complete ACS MS test environment on different
types of cluster. The following components are set up:

* A Postgres database
* Fleet Manager
* Fleetshard Sync
* RHACS Operator

The RHACS operator can be installed from OpenShift marketplace or Quay. Images for Fleet Manager & Fleetshard Sync can either be pulled from Quay or built directly from the source.

##### Required tools
* standard Unix environment with Bash
* `docker` CLI (or replacement)
* Minikube or equivalent (for local deployment)
* operator-sdk (if deploying to clusters not having access to OpenShift Marketplace, like Minikube)
* `yq` & `jq`
* `kubectl` or `oc`
* [chamber](https://github.com/segmentio/chamber#installing)
* [aws CLI](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)

## Scripts

The following scripts exist currently in `dev/env/scripts`:

* `lib.sh`: Basic initialization and library script for the other executable scripts.
* `apply` & `delete`: Convenience scripts for applying and deleting Kubernetes resources supporting environment interpolation.
* `port-forwarding`: Convenient abstraction layer for kubectl port-forwarding.
* `bootstrap.sh`: Sets up the basic environment: creates namespaces, injects image-pull-secrets if necessary, installs OLM (if required), installs RHACS operator (if desired), pulls required images, etc.
* `up.sh`: Brings up the ACS MS environment consisting of the database, `fleet-manager` and `fleetshard-sync`.
* `down.sh`: Deletes the resources created by `up.sh`.

The scripts can be configured using environment variables, the most important options being:

* `CLUSTER_TYPE`: Can be `minikube`, `colima`, `rancher-desktop`, `crc`, `openshift-ci`, `infra-openshift`). Will be
  auto-sensed in most situations depending on the cluster name.
* `FLEET_MANAGER_IMAGE`: Reference for an `acs-fleet-manager` image. If unset, build a fresh image from the current source and deploy that.
* `STATIC_TOKEN`: Needs to contain a valid test user token (can be found in BitWarden)
* `STATIC_TOKEN_ADMIN`: Needs to contain a valid admin token (can be found in BitWarden)
* `QUAY_USER` & `QUAY_TOKEN`: Mandatory setting in case images need to be pulled from Quay.

## Prepare the environment
1. Install the [necessary tools](#Required tools)
2. Login to the AWS using CLI:
    ```shell
    $ aws configure
    AWS Access Key ID [None]: <dev key id>
    AWS Secret Access Key [None]: <dev access key>
    Default region name [None]: us-east-1
    Default output format [None]:
    ```
3. Verify that configuration can be loaded from the Parameter Store
    ```shell
    $ chamber list fleet-manager
    Key                             Version         LastModified            User
    ...

    $ chamber list fleetshard-sync
    Key                             Version         LastModified            User
    ...
    ```
4. Set up a test cluster using [one of the supported](#Cluster setup) types
5. Ensure the `kubectl` context is pointing to the desired cluster:
    ```shell
    kubectl use-context <cluster>
    ```  
6. Set the required environment variables:
   * `QUAY_USER`
   * `QUAY_TOKEN`
   * `STATIC_TOKEN`
   * `STATIC_TOKEN_ADMIN`

## E2E tests

### Full lifecycle
The primary way for executing the e2e test suite is by calling
```shell
$ ./.openshift-ci/test/e2e.sh
```
This will trigger the FULL test lifecycle including the cluster bootstrap, building the image (unless `FLEET_MANAGER_IMAGE` points to a specific image tag), deploying it and running E2E tests.

### Controlling the execution
In certain situations it is also useful to be able to execute the respective building blocks manually:
##### Prepare the cluster
Prepare the cluster by installing the necessary components, such as stackrox-operator and openshift-router
```shell
$ make prepare/dev # points to bootstrap.sh
```
##### Build and deploy
The following command is used for building the Managed Services components image and deploying it on the cluster
```shell
$ make deploy/dev # points to up.sh
```
##### Execute tests
Then, after fleet-manager's leader election is complete (check its logs), you can run the e2e test
suite manually:
```shell
make test/e2e
```
The env var `WAIT_TIMEOUT` can be used to adjust the timeout of each individual tests, using a string compatible with Golang's `time.ParseDuration`, e.g. `WAIT_TIMEOUT=20s`. If not set all tests use 5 minutes as timeout.

##### Cleanup
To clean up the environment run
```shell
$ make undeploy/dev # points to down.sh
```

### DNS tests

The test suite has auto-sensing logic built in to skip DNS e2e tests when the test environment does  not support execution of DNS e2e tests. Currently this is only supported in OpenShift environments.

To run the DNS e2e tests additionally to the default e2e test setup the cluster you're running against needs to have the openshift Route Custom Resource Definition installed and you need to set following environment variables:

```shell
export ROUTE53_ACCESS_KEY="<key-id>"
export ROUTE53_SECRET_ACCESS_KEY="<secret-key>"

# Depending on cluster type and its default configuration you might need
export ENABLE_CENTRAL_EXTERNAL_CERTIFICATE_DEFAULT=true

# If the domain you test against is not the default dev domain
export CENTRAL_DOMAIN_NAME="<domain>"
```


## Cluster setup
Bootstrap a local cluster using one of the options below.

### Minikube

Make sure that Minikube is running with options such as:
```shell
$ minikube start --memory=6G \
                 --cpus=2 \
                 --apiserver-port=8443 \
                 --embed-certs=true \
                 --delete-on-failure=true \
                 --driver=hyperkit # For example
```

and that the `docker` CLI is in `PATH` (if not, export `DOCKER=...` accordingly).

### Colima

Make sure that Colima is running with options such as:
```shell
$ colima start -c 4 -d 60 -m 16 -k
```

and that the `colima` CLI is in `PATH` (if not, export `DOCKER=/path/to/bin/colima nerdctl -- -n k8s.io` accordingly).

### CRC

CRC needs a lot of resources and so does a Central tenant. At least the following resource settings were required to make the test succeed on CRC.

```shell
crc config set memory 18432
crc config set cpus 7
```

There's currently no automated way to upload the fleet-manager image to CRC. Set the `FLEET_MANAGER_IMAGE` environment variable to an available Image in quay or build locally and load it into CRC registry manually.

# Troubleshooting

### Error: Failed to list store contents: MissingRegion: could not find region configuration
This error happens when there's no default AWS region. Run `aws configure` and enter the region properly, or add it manually to the `~/.aws/config`:
```text
[default]
region = us-east-1
```
### Error: Failed to list store contents: NoCredentialProviders: no valid providers in chain. Deprecated.
This error happens when aws CLI is not properly configured. Run `aws configure` and point it to the correct account
