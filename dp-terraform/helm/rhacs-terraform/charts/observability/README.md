# Data plane terraform observability Helm chart

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
  --values ~/acs-terraform-obs-values.yaml .
```

**Uninstall the chart and cleanup all created resources**

```bash
helm uninstall rhacs-terraform-obs --namespace rhacs
```

See internal wiki for an example file `~/.rh/obs-values.yaml`.
