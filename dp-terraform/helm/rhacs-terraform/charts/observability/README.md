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

## Custom resource definitions

The initial deployment of the chart requires the installation of the `Observability`
custom resource definition. It's required to define the customized `observability-stack`
custom resource in the template folder. Helm installs all CRDs inside the `crds/` folder
on the first run. Afterwards OLM and the observability operator itself keep the CRDs up to date.
See the [Helm documentation](https://helm.sh/docs/chart_best_practices/custom_resource_definitions/#some-caveats-and-explanations)
for some caveats and explanations of this approach.

The following commands generate `crds/observability.yaml`:

```
git clone git@github.com:redhat-developer/observability-operator.git
cd observability-operator
git checkout v4.0.4
kustomize build config/crd > crds/observability.yaml
```

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
