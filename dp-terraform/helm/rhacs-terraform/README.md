# Dataplane terraform Helm chart

Chart to terraform dataplane OSD clusters.

## Usage

The env var `FM_ENDPOINT` should point to an endpoint for the fleet manager. An option to use a fleet manager instance running in your laptop is to [setup ngrok](https://ngrok.com/docs/getting-started), launch the fleet manager, and run `ngrok http 8000` to expose it to the internet. That commands outputs an endpoint that you can use for `FM_ENDPOINT`.

Install the chart as follows:

```bash
oc create namespace rhacs
helm -n rhacs install rhacs-terraform dp-terraform/helm/rhacs-terraform/ \
      --set fleetshardSync.ocmToken=$(ocm token) \
      --set fleetshardSync.fleetManagerEndpoint=${FM_ENDPOINT} \
      --set fleetshardSync.clusterId=${cluster_id}
```
