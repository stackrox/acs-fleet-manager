# Data plane terraform Helm chart

Chart to terraform data plane OSD clusters.

## Usage

**Prepare environment variables**

The env var `FM_ENDPOINT` should point to an endpoint for the fleet manager. An option to use a fleet manager instance running in your laptop is to [setup ngrok](https://ngrok.com/docs/getting-started), launch the fleet manager, and run `ngrok http 8000` to expose it to the internet. That commands outputs an endpoint that you can use for `FM_ENDPOINT`.  
To get the cluster id for staging look for `cluster_id` in `dev/config/dataplane-cluster-configuration-staging.yaml` file. Export that value to environment variable `export CLUSTER_ID="<cluster_id from config file>"`.

**Create values file**

Create a file `~/acs-terraform-values.yaml` with the values for the parameters in [values.yaml](./values.yaml) that are missing or that you want to override. That file will contain credentials, so make sure you put it in a safe location, and with suitable permissions.

**Render the chart to see the generated templates during development**

```bash
helm template rhacs-terraform \
  --debug \
  --namespace rhacs \
  --values ~/acs-terraform-values.yaml \
  --set fleetshardSync.ocmToken=$(ocm token --refresh) \
  --set fleetshardSync.fleetManagerEndpoint=${FM_ENDPOINT} \
  --set fleetshardSync.clusterId=${CLUSTER_ID} \
  --set acsOperator.enabled=true .
```

**Install the chart**

```bash
helm install rhacs-terraform \
  --namespace rhacs \
  --create-namespace \
  --values ~/acs-terraform-values.yaml \
  --set fleetshardSync.ocmToken=$(ocm token --refresh) \
  --set fleetshardSync.fleetManagerEndpoint=${FM_ENDPOINT} \
  --set fleetshardSync.clusterId=${CLUSTER_ID} \
  --set acsOperator.enabled=true .
```

**Update the helm release (re-terraform data plane cluster)**

1. Get values used for the latest terraforming
```
helm get values rhacs-terraform --namespace rhacs > ~/re-terraform-dp-cluster-values.yaml
```
2. Adjust values in the values file `~/re-terraform-dp-cluster.yaml` accordingly
3. Check changes with the diff plugin. To install diff plugin please check documentation here: [https://github.com/databus23/helm-diff](https://github.com/databus23/helm-diff)
```
helm diff upgrade rhacs-terraform --namespace rhacs --values ~/re-terraform-dp-cluster-values.yaml .
```
4. Update the helm release
```
helm upgrade rhacs-terraform --namespace rhacs --values ~/re-terraform-dp-cluster-values.yaml .
```

**Uninstall the chart and cleanup all created resources**

```bash
helm uninstall rhacs-terraform --namespace rhacs
```

See internal wiki for an example file `~/.rh/terraform-values.yaml`.
