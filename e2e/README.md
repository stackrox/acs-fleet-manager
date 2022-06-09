# e2e tests

## Run it

```
# Setup a k8s cluster with the RHACS operator running

# Run fleet-manager + database locally
$ ./scripts/setup-dev-env.sh

# Run fleetshard-sync locally
$ make fleetshard-sync
$ CLUSTER_ID=1234567890abcdef1234567890abcdef OCM_TOKEN=$(ocm token) ./fleetshard-sync

# Run e2e tests
$ RUN_E2E=true OCM_TOKEN=$(ocm token) go test ./e2e/...

# To clean up the environment run
$ ./e2e/cleanup.sh
```
