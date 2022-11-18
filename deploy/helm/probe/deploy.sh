#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# shellcheck source=scripts/lib/external_config.sh
source "$SCRIPT_DIR/../../../scripts/lib/external_config.sh"

if [[ $# -ne 2 ]]; then
    echo "Usage: $0 [environment] [cluster]" >&2
    echo "Known environments: stage prod"
    echo "Cluster typically looks like: acs-{environment}-dp-01"
    exit 2
fi

ENVIRONMENT=$1
CLUSTER_NAME=$2

export AWS_PROFILE="$ENVIRONMENT"

init_chamber

load_external_config probe PROBE_

case $ENVIRONMENT in
  stage)
    FM_ENDPOINT="https://api.stage.openshift.com"
    PROBE_IMAGE="quay.io/rhacs-eng/blackbox-monitoring-probe-service:main"
    ;;

  prod)
    FM_ENDPOINT="https://api.openshift.com"
    PROBE_IMAGE="quay.io/rhacs-eng/blackbox-monitoring-probe-service:2b0c84d"
    ;;

  *)
    echo "Unknown environment ${ENVIRONMENT}"
    exit 2
    ;;
esac

function assert_environment() {
  local EXPECTED_ENVIRONMENT="$1"
  if [[ $EXPECTED_ENVIRONMENT != "$ENVIRONMENT" ]]; then
    echo "Cluster ${CLUSTER_NAME} is expected to be in environment ${EXPECTED_ENVIRONMENT}, not ${ENVIRONMENT}" >&2
    exit 2
  fi
}

# The following values can be retrieved from the Red Hat Hybrid Cloud Console.
# - Cluster ID the first piece of information on the "Details" pane of the Overview tab of the given cluster.
# - The URL infix is the part of the Control Plane API endpoint between cluster name and "openshiftapps.com",
#   in the "Cluster ingress" pane of the Networking tab for the given cluster.
case $CLUSTER_NAME in
acs-stage-dp-01)
  assert_environment stage
  ;;
acs-prod-dp-01)
  assert_environment prod
  ;;
*)
  echo "Unknown cluster ${CLUSTER_NAME}. Please define it in the $0 script if this is a new cluster." >&2
  exit 2
  ;;
esac

load_external_config "cluster-${CLUSTER_NAME}" CLUSTER_
oc login --token="${CLUSTER_ROBOT_OC_TOKEN}" --server="https://api.${CLUSTER_NAME}.${CLUSTER_URL_INFIX}.openshiftapps.com:6443"

NAMESPACE="rhacs-probe"
AUTH_TYPE="OCM"

# helm template --debug ... to debug changes
helm upgrade rhacs-probe "${SCRIPT_DIR}" \
  --install \
  --namespace "${NAMESPACE}" \
  --create-namespace \
  --set authType="${AUTH_TYPE}" \
  --set fleetManagerEndpoint="${FM_ENDPOINT}" \
  --set image="${PROBE_IMAGE}" \
  --set ocm.token="${PROBE_OCM_TOKEN}" \
  --set ocm.username="${PROBE_OCM_USERNAME}" \
  --set pullSecret="${PROBE_PULL_SECRET}"

# To uninstall an existing release:
# helm uninstall rhacs-probe --namespace rhacs-probe
#
# To delete all resources specified by the template:
# helm template ... > /var/tmp/resources.yaml
# kubectl delete -f /var/tmp/resources.yaml
