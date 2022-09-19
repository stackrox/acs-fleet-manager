#!/usr/bin/env bash
set -eo pipefail

CLUSTER_ID="test-clusterId"
FM_ENDPOINT="127.0.0.1:443"
OCM_TOKEN="example-token"

helm template rhacs-terraform \
  --debug \
  --namespace rhacs \
  --set fleetshardSync.ocmToken=${OCM_TOKEN} \
  --set fleetshardSync.fleetManagerEndpoint=${FM_ENDPOINT} \
  --set fleetshardSync.clusterId=${CLUSTER_ID} \
  --set acsOperator.enabled=true .
