# Using the Fleet Manager service

## Fleet Manager Environments

The service can be run in a number of different environments. Environments are
essentially bespoke sets of configuration that the service uses to make it
function differently. Environments can be set using the `OCM_ENV` environment
variable. Below are the list of known environments and their
details.

- `development` (default) - The `staging` OCM environment is used.
  Debugging utilities are enabled. This should be used in local development.
  The `OCM_ENV` variable has not been set.
- `testing` - The OCM API is mocked/stubbed out, meaning network calls to OCM
  will fail. The auth service is mocked. This should be used for unit testing.
- `integration` - Identical to `testing` but using an emulated OCM API server
  to respond to OCM API calls, instead of a basic mock. This can be used for
  integration testing to mock OCM behaviour.
- `production` - Debugging utilities are disabled.
  This environment can be ignored in most development and is only used when
  the service is deployed.

The `OCM_ENV` environment variable should be set before running any Fleet
Manager binary command or Makefile target

## Running Fleet Manager locally

### Running the fleet manager with an OSD cluster from infractl

Write a Cloud provider configuration file that matches the cloud provider and region used for the cluster, see `dev/config/provider-configuration-infractl-osd.yaml` for an example OSD cluster running in GCP. See the cluster creation logs in https://infra.rox.systems/cluster/YOUR_CLUSTER to locate the provider and region. See `internal/central/pkg/services/cloud_providers.go` for the provider constant.

Enable a cluster configuration file for the OSD cluster, see `dev/config/dataplane-cluster-configuration-infractl-osd.yaml` for an example OSD cluster running in GCP. Again, see the cluster creation logs for possibly missing required fields.

Launch the fleet manager using those configuration files:

```bash
make binary && ./fleet-manager serve \
   --dataplane-cluster-config-file=$(pwd)/dev/config/dataplane-cluster-configuration-infractl-osd.yaml \
   --providers-config-file=$(pwd)/dev/config/provider-configuration-infractl-osd.yaml \
   2>&1 | tee fleet-manager-serve.log
```

## Running containerized fleet-manager and fleetshard-sync on a test cluster

A test cluster can be either local or remote. Recommended local clusters are _colima_ for macOS and _kind_ for linux. The easiest way to provision a remote OpenShift cluster is to use StackRox Infra and the infractl tool.
As an alternative, refer to this [guide](./setup-developer-osd-cluster.md) to set up an OSD cluster yourself.
The makefile target `image/build` builds a combined image, containing both applications, `fleet-manager` and `fleetshard-sync`.
To deploy the image please refer to the guide: [setup-test-environment.md](./setup-test-environment.md)
