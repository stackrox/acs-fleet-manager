# External probe

The probe service enables blackbox monitoring for fleet manager. During each
probe run, it attempts to

1. create a Central instance and ensure it is in `ready` state.
2. verify that the Central instance is of type `standard` and that the Central UI is reachable.
3. deprovision the Central.

Requests against fleet manager are authenticated by a Red Hat SSO service account.
A single probe can be executed with `probe run`. If the probe aborts or fails, the service exits with exit code 1.
To periodically collect probes it can be started as a daemon with `probe start`. Results of the probes are exposed on a Prometheus metrics endpoint `PROBE_METRICS_ADDRESS`.
When receiving an interrupt signal, a graceful shutdown cleans up remaining resources.

## Quickstart

Execute all commands from git root directory.

1. Set up a dataplane configuration file in `./$CLUSTER_ID.yaml`. See [](../config/dataplane-cluster-configuration.yaml) for an example.
2. Create a service account for the probe service via the [OpenShift console](https://console.redhat.com/application-services/service-accounts).
3. Assign quota to the service account via the [quota list](../config/quota-management-list-configuration.yaml).
4.

```sh
# Start fleet manager
/fleet-manager serve --dataplane-cluster-config-file "./$CLUSTER_ID.yaml"

# Set environment variables
export RHSSO_SERVICE_ACCOUNT_CLIENT_ID=<service-account-client-id>
export RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET=<service-account-client-secret>

# Build the binary
make probe

# Start the probe service and run a single probe
./probe/bin/probe run

# or run an endless loop of probes
./probe/bin/probe start
```
