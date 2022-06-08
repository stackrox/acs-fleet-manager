# fleetshard-sync

## Prerequisites

The sample configuration for a dataplane cluster used in the instructions below requires Minikube to be running.

Therefore, make sure that Minikube is running:
```
$ minikube start
```
(specific settings depend on your environment, see e.g. https://minikube.sigs.k8s.io/docs/drivers/)

Proceed by running the RHACS operator:
```
$ cdrox
$ make -C operator install run
```

## Quickstart

```
```
# Start commands from git root directory

# Start fleet manager
$ ./scripts/setup-dev-env.sh

# Build and run fleetshard-sync
$ make fleetshard/build
$ OCM_TOKEN=$(ocm token) CLUSTER_ID=1234567890abcdef1234567890abcdef ./fleetshard-sync

# Create a central instace
$ ./scripts/create-central.sh
```
