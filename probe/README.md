# External probe

The probe service enables blackbox monitoring for fleet manager. During each
probe run, it attempts to

1. create a Central instance and ensure it is in `ready` state.
2. verify that the Central instance is of type `standard` and that the Central UI is reachable.
3. deprovision the Central.

Requests against fleet manager are authenticated by a Red Hat SSO service account.

The probe service may run as a one-shot command (`probe run`), in which case a single probe
is executed. If the probe aborts or fails, the service exits with exit code 1. In
addition, the probe service may run in a continuous loop (`probe start`), in which case
probe runs are executed in an endless loop. Prometheus metrics capture the results of the probes.

The probe service implements graceful shutdown, which means upon receiving an interrupt signal, it
attempts to clean up remainig resources.

## Quickstart

Execute all commands from git root directory.

1. Set up a dataplane configuration file in `./$CLUSTER_ID.yaml`.
2. Create a service account for the probe service via the [OpenShift console](https://console.redhat.com/application-services/service-accounts).
3. Assign quota to the service account via the [quota list](../config/quota-management-list-configuration.yaml).
4. Start fleet manager

```sh
/fleet-manager serve --dataplane-cluster-config-file "./$CLUSTER_ID.yaml"
```

5. Set environment variables

```sh
export RHSSO_SERVICE_ACCOUNT_CLIENT_ID=<service-account-client-id>
export RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET=<service-account-client-secret>
```

6. Build the binary

```sh
make probe
```

7. Start the probe service and run a single probe

```sh
./probe/bin/probe run
```

or run a endless loop of probes

```sh
./probe/bin/probe start
```
