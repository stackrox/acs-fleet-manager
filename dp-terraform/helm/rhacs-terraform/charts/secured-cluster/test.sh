#!/usr/bin/env bash
set -eo pipefail

SECURED_CLUSTER_CLUSTER_NAME="test-cluster"
SECURED_CLUSTER_CENTRAL_ENDPOINT="central-endpoint"
SECURED_CLUSTER_CA_CERT="$(yq .ca.cert init-bundle.yaml)"
SECURED_CLUSTER_ADMISSION_CONTROL_CERT="$(yq .admissionControl.serviceTLS.cert init-bundle.yaml)"
SECURED_CLUSTER_ADMISSION_CONTROL_KEY="$(yq .admissionControl.serviceTLS.key init-bundle.yaml)"
SECURED_CLUSTER_COLLECTOR_CERT="$(yq .collector.serviceTLS.cert init-bundle.yaml)"
SECURED_CLUSTER_COLLECTOR_KEY="$(yq .collector.serviceTLS.key init-bundle.yaml)"
SECURED_CLUSTER_SENSOR_CERT="$(yq .sensor.serviceTLS.cert init-bundle.yaml)"
SECURED_CLUSTER_SENSOR_KEY="$(yq .sensor.serviceTLS.key init-bundle.yaml)"

helm template secured-cluster \
  --debug \
  --include-crds \
  --namespace rhacs \
  --set clusterName="${SECURED_CLUSTER_CLUSTER_NAME}" \
  --set centralEndpoint="${SECURED_CLUSTER_CENTRAL_ENDPOINT}" \
  --set ca.cert="${SECURED_CLUSTER_CA_CERT}" \
  --set admissionControl.serviceTLS.cert="${SECURED_CLUSTER_ADMISSION_CONTROL_CERT}" \
  --set admissionControl.serviceTLS.key="${SECURED_CLUSTER_ADMISSION_CONTROL_KEY}" \
  --set collector.serviceTLS.cert="${SECURED_CLUSTER_COLLECTOR_CERT}" \
  --set collector.serviceTLS.key="${SECURED_CLUSTER_COLLECTOR_KEY}" \
  --set sensor.serviceTLS.cert="${SECURED_CLUSTER_SENSOR_CERT}" \
  --set sensor.serviceTLS.key="${SECURED_CLUSTER_SENSOR_KEY}" \
  .
