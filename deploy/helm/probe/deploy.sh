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
PROBE_IMAGE_ORG="rhacs-eng"
PROBE_IMAGE_NAME="blackbox-monitoring-probe-service"
# Get the first non-merge commit, starting with HEAD.
# On main this should be HEAD, on production, the latest merged main commit.
PROBE_IMAGE_TAG="$(git rev-list --no-merges --max-count 1 --abbrev-commit --abbrev=7 HEAD)"
PROBE_IMAGE="quay.io/${PROBE_IMAGE_ORG}/${PROBE_IMAGE_NAME}:${PROBE_IMAGE_TAG}"

export AWS_PROFILE="$ENVIRONMENT"

init_chamber

load_external_config probe PROBE_

case $ENVIRONMENT in
  stage)
    FM_ENDPOINT="https://api.stage.openshift.com"
    ;;

  prod)
    FM_ENDPOINT="https://api.openshift.com"
    ;;

  *)
    echo "Unknown environment ${ENVIRONMENT}"
    exit 2
    ;;
esac

CLUSTER_ENVIRONMENT="$(echo "${CLUSTER_NAME}" | cut -d- -f 2)"
if [[ $CLUSTER_ENVIRONMENT != "$ENVIRONMENT" ]]; then
    echo "Cluster ${CLUSTER_NAME} is expected to be in environment ${CLUSTER_ENVIRONMENT}, not ${ENVIRONMENT}" >&2
    exit 2
fi

if [[ "${HELM_PRINT_ONLY:-}" == "true" ]]; then
    HELM_DEBUG_FLAGS="--debug --dry-run"
else
    "${SCRIPT_DIR}/../../../scripts/check_image_exists.sh" "${PROBE_IMAGE_ORG}" "${PROBE_IMAGE_NAME}" "${PROBE_IMAGE_TAG}"
fi

load_external_config "cluster-${CLUSTER_NAME}" CLUSTER_
oc login --token="${CLUSTER_ROBOT_OC_TOKEN}" --server="$CLUSTER_URL"

NAMESPACE="rhacs-probe"
AUTH_TYPE="OCM"

# helm template --debug ... to debug changes
helm upgrade rhacs-probe "${SCRIPT_DIR}" "${HELM_DEBUG_FLAGS:-}" \
  --install \
  --namespace "${NAMESPACE}" \
  --create-namespace \
  --set authType="${AUTH_TYPE}" \
  --set clusterName="${CLUSTER_NAME}" \
  --set environment="${ENVIRONMENT}" \
  --set fleetManagerEndpoint="${FM_ENDPOINT}" \
  --set image="${PROBE_IMAGE}" \
  --set ocm.token="${PROBE_OCM_TOKEN}" \
  --set ocm.username="${PROBE_OCM_USERNAME}"

# To uninstall an existing release:
# helm uninstall rhacs-probe --namespace rhacs-probe
#
# To delete all resources specified by the template:
# helm template ... > /var/tmp/resources.yaml
# kubectl delete -f /var/tmp/resources.yaml
