# Dataplane terraform Helm chart

Chart to terraform dataplane OSD clusters.

## Usage

The env var `FM_ENDPOINT` should point to an endpoint for the fleet manager. An option to use a fleet manager instance running in your laptop is to [setup ngrok](https://ngrok.com/docs/getting-started), launch the fleet manager, and run `ngrok http 8000` to expose it to the internet. That commands outputs an endpoint that you can use for `FM_ENDPOINT`.  
To get the cluster id for staging use `cluster_id=$(grep cluster_id ../../../dev/config/dataplane-cluster-configuration-staging.yaml | tail -n1 | tr -s ' '  | cut -d ' ' -f 3)`

Create a file `obs-values.yaml` with the values for the parameters in [values.yaml](./values.yaml) that are missing or that you want to override. That file will contain credentials, so make sure you put it in a safe location, and with suitable permissions. 

The Makefile in this directory has targets for typical tasks:

- Render the chart to see the generated templates during development: 

```bash
make helm/render ns=rhacs values=~/.rh/terraform-values.yaml helm_args="\
   --set fleetshardSync.ocmToken=$(ocm token) \
   --set fleetshardSync.fleetManagerEndpoint=${FM_ENDPOINT} \
   --set fleetshardSync.clusterId=${cluster_id} \
   --set acsOperator.enabled=true"
```

- Install the chart: 

```bash
make install ns=rhacs values=~/.rh/terraform-values.yaml helm_args="\
   --set fleetshardSync.ocmToken=$(ocm token) \
   --set fleetshardSync.fleetManagerEndpoint=${FM_ENDPOINT} \
   --set fleetshardSync.clusterId=${cluster_id} \
   --set acsOperator.enabled=true"
```

- Uninstall the chart and cleanup all created resouces: `make uninstall ns=rhacs`.

See internal wiki for an example file `~/.rh/terraform-values.yaml`.