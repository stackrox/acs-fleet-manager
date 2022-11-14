#!/usr/bin/env bash
set -eo pipefail

# Render only operator related manifests.

CLUSTER_ID="test-clusterId"
FM_ENDPOINT="127.0.0.1:443"
OCM_TOKEN="example-token"

helm template rhacs-terraform \
  --debug \
  --namespace rhacs \
  --set fleetshardSync.ocmToken=${OCM_TOKEN} \
  --set fleetshardSync.fleetManagerEndpoint=${FM_ENDPOINT} \
  --set fleetshardSync.clusterId=${CLUSTER_ID} \
  --set acsOperator.enabled=true . \
  --set acsOperator.source=rhacs-operator-source \
  --set acsOperator.upstream=true \
  --set acsOperator.version=v3.72.0	\
  -s templates/acs-operator.yaml
