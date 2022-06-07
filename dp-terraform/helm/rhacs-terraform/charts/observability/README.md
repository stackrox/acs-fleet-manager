# Dataplane terraform observability Helm chart

## Usage

Create a file `obs-values.yaml` with the values for the parameters in [values.yaml](./values.yaml) that are missing or that you want to override. That file will contain credentials, so make sure you put it in a safe location, and with suitable permissions. 

Render the chart to see the generated templates during development:

```bash
helm -n testn template rhacs-terraform-obs dp-terraform/helm/rhacs-terraform/charts/observability --debug -f ~/.rh/obs-values.yaml
```


Install the chart:

```bash
oc create namespace rhacs
helm -n rhacs install rhacs-terraform-obs dp-terraform/helm/rhacs-terraform/charts/observability -f ~/.rh/obs-values.yaml
helm -n rhacs list
```


TODO install from top

---

