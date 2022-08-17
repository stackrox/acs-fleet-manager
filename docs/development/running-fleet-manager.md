## Using the Fleet Manager service

### Fleet Manager Environments

The service can be run in a number of different environments. Environments are
essentially bespoke sets of configuration that the service uses to make it
function differently. Environments can be set using the `OCM_ENV` environment
variable. Below are the list of known environments and their
details.

- `development` (default) - The `staging` OCM environment is used. Sentry is disabled.
  Debugging utilities are enabled. This should be used in local development.
  The `OCM_ENV` variable has not been set.
- `testing` - The OCM API is mocked/stubbed out, meaning network calls to OCM
  will fail. The auth service is mocked. This should be used for unit testing.
- `integration` - Identical to `testing` but using an emulated OCM API server
  to respond to OCM API calls, instead of a basic mock. This can be used for
  integration testing to mock OCM behaviour.
- `production` - Debugging utilities are disabled, Sentry is enabled.
  This environment can be ignored in most development and is only used when
  the service is deployed.

The `OCM_ENV` environment variable should be set before running any Fleet
Manager binary command or Makefile target

### Running the fleet manager with an OSD cluster from infractl

Write a Cloud provider configuration file that matches the cloud provider and region used for the cluster, see `dev/config/provider-configuration-infractl-osd.yaml` for an example OSD cluster running in GCP. See the cluster creation logs in https://infra.rox.systems/cluster/YOUR_CLUSTER to locate the provider and region. See `internal/dinosaur/pkg/services/cloud_providers.go` for the provider constant.

Enable a cluster configuration file for the OSD cluster, see `dev/config/dataplane-cluster-configuration-infractl-osd.yaml` for an example OSD cluster running in GCP. Again, see the cluster creation logs for possibly missing required fields.

Download the kubeconfig for the cluster. Without this the fleet manager will refuse to use the cluster.

```bash
CLUSTER=... # your cluster's name
infractl artifacts "${CLUSTER}" --download-dir "~/infra/${CLUSTER}"
```

Launch the fleet manager using those configuration files:

```bash
make binary && ./fleet-manager serve \
   --dataplane-cluster-config-file=$(pwd)/dev/config/dataplane-cluster-configuration-infractl-osd.yaml \
   --providers-config-file=$(pwd)/dev/config/provider-configuration-infractl-osd.yaml \
   --kubeconfig="~/infra/${CLUSTER}/kubeconfig" \
   2>&1 | tee fleet-manager-serve.log
```

### Running containerized fleet-manager and fleetshard-sync

The makefile target `image/build` builds a combined image, containing both applications, `fleet-manager` and `fleetshard-sync`.

So far only `fleet-manager` can be successfully spawned from this image, because `fleetshard-sync` tries to reach `fleet-manager` at `127.0.0.1` (hard-coded).

Using e.g. the Docker CLI, `fleet-manager` can be spawned as follows:

```
docker run -it --rm -p 8000:8000 \
   -v "$(git rev-parse --show-toplevel)/config":/config \
   -v "$(git rev-parse --show-toplevel)/secrets":/secrets \
   <IMAGE REFERENCE> \
   --db-host-file secrets/db.host.internal-docker \
   --api-server-bindaddress 0.0.0.0:8000
```

Using the above command the `fleet-manager` application tries to access its database running on the host system and its API server is
reachable at `localhost` (host system).

In principle `fleetshard-sync` will be able to spawned using a command similar to the following:

```
OCM_TOKEN=$(ocm token --refresh)
docker run -it -e OCM_TOKEN --rm -p 8000:8000 \
   --entrypoint /usr/local/bin/fleetshard-sync \
   -v "$(git rev-parse --show-toplevel)/config":/config \
   -v "$(git rev-parse --show-toplevel)/secrets":/secrets \
   <IMAGE REFERENCE>
```

For this to work `fleetshard-sync` has to be modified so that `fleet-manager`'s endpoint is configurable and both containers have to be
running using a shared network so that they can access each other (TODO).
