# Data plane terraform secured-cluster Helm chart

This chart simply installs a `SecuredCluster` CR and the dependent secrets
required to authenticate against a Central instance.

## Custom resource definitions

The initial deployment of the chart requires the installation of the `SecuredCluster`
custom resource definition. It's required to define the customized `stackrox-secured-cluster-services`
custom resource in the template folder. Helm installs all CRDs inside the `crds/` folder
on the first run. Afterwards OLM and the observability operator itself keep the CRDs up to date.
See the [Helm documentation](https://helm.sh/docs/chart_best_practices/custom_resource_definitions/#some-caveats-and-explanations)
for some caveats and explanations of this approach.

The following commands generate `crds/secured-cluster.yaml`:

```
git clone git@github.com:stackrox/stackrox.git
cd stackrox/operator
git checkout 3.74.0
kustomize build config/crd > crds/secured-cluster.yaml
```

The `centrals.platform.stackrox.io` CRD will need to be deleted from the output file.

## Usage


Create a file `~/acs-terraform-secured-cluster-values.yaml` with the values for the parameters in [values.yaml](./values.yaml) that are missing or that you want to override. That file will contain credentials, so make sure you put it in a safe location, and with suitable permissions.

**Render the chart to see the generated templates during development**

```bash
helm template secured-cluster \
  --debug \
  --namespace rhacs \
  --values ~/acs-terraform-obs-values.yaml .
```

**Install or update the chart**

```bash
helm upgrade --install secured-cluster \
  --namespace rhacs \
  --create-namespace \
  --values ~/acs-terraform-obs-values.yaml .
```

**Uninstall the chart and cleanup all created resources**

```bash
helm uninstall secured-cluster --namespace rhacs
```
