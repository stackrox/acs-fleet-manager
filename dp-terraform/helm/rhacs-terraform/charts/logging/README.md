# Data plane terraform logging Helm chart

This chart installs resource into `openshift-logging` namespace. This namespace is Openshift dedicated namespace for logging stack for OSD cluster.
It installs on top the openshift eventrouter in order to log kubernetes events in the `openshift-logging` namespace.

## Custom resource definitions

The initial deployment of the chart requires the installation of the `ClusterLogging`
and `ClusterLogForwarder` custom resource definitions. They're required to define the
logging configuration in the template folder. Helm installs all CRDs inside the `crds/`
folder on the first run. Afterwards OLM and the logging operator itself keep the CRDs
up to date. See the
[Helm documentation](https://helm.sh/docs/chart_best_practices/custom_resource_definitions/#some-caveats-and-explanations)
for some caveats and explanations of this approach.

The following commands generate `crds/logging.yaml`:

```
git clone git@github.com/openshift/cluster-logging-operator.git
cd cluster-logging-operator
git checkout release-5.6
kustomize build config/crd > crds/logging.yaml
```

## Usage

Create a file `~/acs-terraform-logging-values.yaml` with the values for the parameters in [values.yaml](./values.yaml) that are missing or that you want to override. That file will contain credentials, so make sure you put it in a safe location, and with suitable permissions.

**Render the chart to see the generated templates during development**

```bash
helm template rhacs-terraform-logging \
  --debug \
  --namespace rhacs \
  --values ~/acs-terraform-logging-values.yaml .
```

**Install or update the chart**

```bash
helm upgrade --install rhacs-terraform-logging \
  --namespace rhacs \
  --create-namespace \
  --values ~/acs-terraform-logging-values.yaml .
```

**Uninstall the chart and cleanup all created resources**

```bash
helm uninstall rhacs-terraform-logging --namespace rhacs
```

**NOTE:** The custom resource definitions created by logging operator will not be removed.
