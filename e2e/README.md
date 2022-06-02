# e2e tests

## Run it

```
# Setup a k8s cluster with the RHACS operator running

# Run fleet-manager + database locally
$ ./scripts/setup-dev-env.sh

# Run fleetshard-sync locally
$ make fleetshard/build
$ OCM_TOKEN=$(ocm token) ./fleetshard-sync

$ export OCM_TOKEN=$(ocm token)
$ go test ./e2e/...

# To clean up the environment run:
$ ./e2e/cleanup.sh
```
