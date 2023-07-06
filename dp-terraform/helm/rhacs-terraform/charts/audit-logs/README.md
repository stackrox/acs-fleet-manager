# Data plane terraform audit-logs Helm chart

This chart installs resource into `rhacs-audit-logs` namespace.

## Usage

Create a file `~/dp-terraform-audit-logs-values.yaml` with the values for the parameters in [values.yaml](./values.yaml) that are missing or that you want to override.

**Render the chart to see the generated templates during development**

```bash
helm template rhacs-terraform-audit-logs \
  --debug \
  --namespace rhacs \
  --values ~/dp-terraform-audit-logs-values.yaml .
```

**Install or update the chart**

```bash
helm upgrade --install rhacs-terraform-audit-logs \
  --namespace rhacs \
  --create-namespace \
  --values ~/dp-terraform-audit-logs-values.yaml .
```

**Uninstall the chart and cleanup all created resources**

```bash
helm uninstall rhacs-terraform-audit-logs --namespace rhacs
```
