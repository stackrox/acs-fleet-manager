# e2e tests

## Run it

```
# Setup a k8s cluster
# Start RHACS operator

# Run fleet-manager + database locally
$ ./scripts/setup-dev-env.sh

# Run fleetshard-sync locally
$ make fleetshard-sync
$ CLUSTER_ID=1234567890abcdef1234567890abcdef OCM_TOKEN=$(ocm token --refresh) ./fleetshard-sync

# Run e2e tests (invalidate cache to run it multiple times)
$ go clean -testcache && CLUSTER_ID=1234567890abcdef1234567890abcdef RUN_E2E=true OCM_TOKEN=$(ocm token --refresh) go test ./e2e/...

# To clean up the environment run
$ ./e2e/cleanup.sh
```

The following env vars can also be adjusted for using a different types of data plane clusters. If not set the test will assume a local minikube cluster:

- `DP_CLOUD_PROVIDER`: cloud provider for the data plane cluster.
- `DP_REGION`: region for the data plane cluster.
- `CLUSTER_ID`: name of the cluster with fleetshard-sync in the data plane.

In addition, you can specify how test should connect to the fleet-manager instance with the following environment variables:

- `FLEET_MANAGER_ENDPOINT`: base fleet-manager url for API calls.
- `OCM_TOKEN`: OCM token for fleet-manager auth api calls.


The env var `WAIT_TIMEOUT` can be used to adjust the timeout of each individual tests, using a string compatible with Golang's `time.ParseDuration`, e.g. `WAIT_TIMEOUT=20s`. If not set all tests use 5 minutes as timeout.
