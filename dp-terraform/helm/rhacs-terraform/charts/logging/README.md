# Data plane terraform logging Helm chart

This chart installs resource into `openshift-logging` namespace. This namespace is Openshift dedicated namespace for logging stack for OSD cluster.

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
