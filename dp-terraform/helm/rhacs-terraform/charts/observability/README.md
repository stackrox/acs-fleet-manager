# Data plane terraform observability Helm chart

## Usage

Create a file `obs-values.yaml` with the values for the parameters in [values.yaml](./values.yaml) that are missing or that you want to override. That file will contain credentials, so make sure you put it in a safe location, and with suitable permissions. 

**Render the chart to see the generated templates during development**

```bash
helm template rhacs-terraform-obs \
  --debug \
  --namespace rhacs \
  --values ~/.rh/obs-values.yaml .
```

**Install the chart**

```bash
helm install rhacs-terraform-obs \
  --namespace rhacs \
  --values ~/.rh/obs-values.yaml .
```

**Uninstall the chart and cleanup all created resources**

```bash
helm uninstall rhacs-terraform-obs --namespace rhacs
```

See internal wiki for an example file `~/.rh/obs-values.yaml`.
