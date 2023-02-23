# Data plane terraform observability Helm chart

## Configuration

The [observability resources repository](https://github.com/stackrox/rhacs-observability-resources) configures
monitoring rules, alertings rules and dashboards by encoding them as Kubernetes custom resources. The
[observability operator](https://github.com/redhat-developer/observability-operator) pulls these resources
from the GitHub repository at `github.tag` and reconciles the resources on the data plane clusters.

The observability operator offers further integrations with
- Observatorium for long term metrics storage.
- PagerDuty for alert routing.
- webhooks for a dead man switch in case the monitoring system degrades.

## Usage

Create a file `~/acs-terraform-obs-values.yaml` with the values for the parameters in [values.yaml](./values.yaml) that are missing or that you want to override. That file will contain credentials, so make sure you put it in a safe location, and with suitable permissions.

**Render the chart to see the generated templates during development**

```bash
helm template rhacs-terraform-obs \
  --debug \
  --namespace rhacs \
  --values ~/acs-terraform-obs-values.yaml .
```

**Install or update the chart**

```bash
helm upgrade --install rhacs-terraform-obs \
  --namespace rhacs \
  --create-namespace \
  --values ~/acs-terraform-obs-values.yaml .
```

**Uninstall the chart and cleanup all created resources**

```bash
helm uninstall rhacs-terraform-obs --namespace rhacs
```

See internal wiki for an example file `~/acs-terraform-obs-values.yaml`.
