# Blackbox monitoring probe service Helm chart

## Usage

Create a file `~/acs-probe-values.yaml` with the values for the parameters in [values.yaml](./values.yaml) that are missing or that you want to override. That file contains credentials, so make sure you put it in a safe location, and with suitable permissions.

**Render the chart to see the generated templates during development**

```bash
helm template rhacs-probe \
  --debug \
  --namespace rhacs-probe \
  --values ~/acs-probe-values.yaml .
```

**Install or update the chart**

```bash
helm upgrade --install rhacs-probe \
  --namespace rhacs-probe \
  --create-namespace \
  --values ~/acs-probe-values.yaml .
```

**Uninstall the chart and cleanup all created resources**

```bash
helm uninstall rhacs-probe --namespace rhacs-probe
```

To remove every resource from the cluster, delete the namespace `rhacs-probe`.
