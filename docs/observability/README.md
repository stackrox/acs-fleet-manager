# Fleet Manager Metrics and Dashboards
This README will outline the adaptations and modifications that need to be made to utilise the fleet manager template metrics and dashboards.

## Template Metrics
The file [metrics.go](../../pkg/metrics/metrics.go) creates Prometheus metrics of differing types. These are the metrics which are then reported and visualised in each Grafana dashboard.
See [here](https://prometheus.io/docs/concepts/metric_types/) for more info about Prometheus metric types

These metrics are grouped by metric subject: data plane clusters, service ('dinosaur' for this template) and reconcilers. These metrics need to be updated with service name (ie 'dinosaur' replaced and service name included).

There is also a section of metrics regarding Observatorium API. These metrics are not reported to a Grafana dashboard by this template. See [kas-fleet-manager metrics configmap](https://gitlab.cee.redhat.com/service/app-interface/-/blob/master/resources/observability/grafana/grafana-dashboard-kas-fleet-manager-metrics.configmap.yaml#L5460-7247) for an example of a Grafana dashboard reporting these metrics.

The file [metrics_middleware.go](../../pkg/handlers/metrics_middleware.go) creates metrics concerned with incoming API requests. This file contains useful and important information about how these metrics are written and reported.

## Grafana Dashboards
See the [Getting Started](https://grafana.com/docs/grafana/latest/getting-started/?pg=docs) section of the Grafana website for general information about Grafana and its uses.

See the [JSON Model](https://grafana.com/docs/grafana/latest/dashboards/json-model/?pg=docs) section for more info about Grafana Dashboard JSON models

### Dashboard fields to be updated
In general `dinosaur` has been used as placeholder throughout this template and should be replaced with the service being used.

There are three `config.yaml` files located in `observability` folder responsible for generating Grafana dashboards:
* `grafana-dashboard-fleet-manager-metrics.configmap.yaml`
* `grafana-dashboard-fleet-manager-stage-slos.configmap.yaml`
* `grafana-dashboard-fleet-manager-prod-slos.configmap.yaml`

Each file contains the JSON data required for the metrics dashboards. Each dashboard consists of panels. For each panel, within `"targets"` arrays, the `"expr"` fields need to be modified to represent the service being used with the fleet manager. Attention should be paid to labels such as `job`, `namespace`, `exported_namespace` within the `"expr"` field.
The `"expr"` fields contain Prometheus queries. The [Querying section](https://prometheus.io/docs/prometheus/latest/querying/basics/) of the Prometheus website has more information about queries.

> NOTE: Make sure to change the `namespace` label so that it corresponds to the namespace where your service will be deployed.

The `"title"`, `"transformations"`, `"legendFormat"` fields may also need to be updated to include the service name. Be aware that each panel can have multiple sub-panels each needing to be adapted.

The dashboard `"uid"` needs to be updated also. See [here](https://grafana.com/docs/grafana/latest/http_api/dashboard/) for more info about dashboard uid.

At the end of each file, outside of the JSON data, the `name` and `grafana-folder` fields require an update.

## Further information
See [SLOs README](../slos/README.md) for more informtion about metrics and their use in measuring SLIs.

> NOTE this document contains references to Red Hat internal components

See [here](https://gitlab.cee.redhat.com/service/app-interface#add-a-grafana-dashboard) for information about adding Grafana dashboards in App-Sre

## Observatorium

The service that the Fleet Manager manages (Dinosaur in the case of this template)
can send metrics to a [Observatorium](https://github.com/observatorium/observatorium)
instance from the data plane. Fleet Manager is also able to interact directly
with Observatorium to retrieve the metrics sent by the managed
service (Dinosaur) from the data plane.

### Configuring Observatorium

To configure a new managed service to use Observatorium a Red Hat managed
Observatorium service can be used. For that, a new `Observatorium Tenant` has
to be created and configured in that Red Hat managed Observatorium service. That
task is done by the Red Hat Observability team. To do so there's
an Onboarding process. See the[Onboarding a Tenant into Red Hat’s Observatorium Instance](https://docs.google.com/document/d/1pjM9RRvij-IgwqQMt5q798B_4k4A9Y16uT2oV9sxN3g) document on how to do it.

If you have any doubts about the onboarding process, The Red Hat Observability
team can be contacted on the #forum-observatorium Slack channel.

## Observability stack

When a data plane cluster is created/assigned in Fleet Manager, the Observability stack is installed as part of the [cluster
Terraforming process](../implementation.md).

The observability stack includes:
* [Observability Operator](https://github.com/redhat-developer/observability-operator): The Observability Operator deploys & maintains a common platform for Application Services to share and utilize to aid in monitoring & reporting on their service components. It integrates with the Observatorium project for pushing metrics and logs to a central location. See the linked repository on details about what is deployed / configured by the operator
* Configuration to set up the Observability stack through Observability Operator. This
  configuration is done by hosting a set of configuration files in a git remote repository that has to be provided as part of
  the Fleet Manager configuration. An example of a git remote repository containing the observability stack configuration in a git remote
  repository for the Managed Kafka service can be found in the [observability-resources-mk](https://github.com/bf2fc6cc711aee1a0c2a/observability-resources-mk) git repository

To provide the parameters to set up the observability stack in fleet manager those are set as
CLI flags to the kas-fleet-manager code. See the [Observability section in the feature-flags documentation file](../feature-flags.md#Observability)
for details.