#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# shellcheck source=scripts/lib/external_config.sh
source "$SCRIPT_DIR/../../../scripts/lib/external_config.sh"
# shellcheck source=scripts/lib/helm.sh
source "$SCRIPT_DIR/../../../scripts/lib/helm.sh"

if [[ $# -ne 2 ]]; then
    echo "Usage: $0 [environment] [cluster]" >&2
    echo "Known environments: dev integration stage prod"
    echo "Cluster typically looks like: acs-{environment}-dp-01"
    exit 2
fi

ENVIRONMENT=$1
CLUSTER_NAME=$2
PROBE_IMAGE_ORG="rhacs-eng"
PROBE_IMAGE_NAME="blackbox-monitoring-probe-service"
# Get HEAD for both main and production. This is the latest merged commit.
PROBE_IMAGE_TAG="$(git rev-parse --short=7 HEAD)"
PROBE_IMAGE="quay.io/${PROBE_IMAGE_ORG}/${PROBE_IMAGE_NAME}:${PROBE_IMAGE_TAG}"

init_chamber

load_external_config probe PROBE_

case $ENVIRONMENT in
  dev)
    FM_ENDPOINT="https://api.fake.openshift.com"
    ;;

  integration)
    FM_ENDPOINT="https://api.integration.openshift.com"
    ;;

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

CLUSTER_ENVIRONMENT="$(echo "${CLUSTER_NAME}" | cut -d- -f 2 | sed 's,^int$,integration,')"
if [[ $CLUSTER_ENVIRONMENT != "$ENVIRONMENT" ]]; then
    echo "Cluster ${CLUSTER_NAME} is expected to be in environment ${CLUSTER_ENVIRONMENT}, not ${ENVIRONMENT}" >&2
    exit 2
fi

if [[ "${HELM_DRY_RUN:-}" == "true" ]]; then
    "${SCRIPT_DIR}/../../../scripts/check_image_exists.sh" "${PROBE_IMAGE_ORG}" "${PROBE_IMAGE_NAME}" "${PROBE_IMAGE_TAG}" 0 || echo >&2 "Ignoring failed image check in dry-run mode."
else
    "${SCRIPT_DIR}/../../../scripts/check_image_exists.sh" "${PROBE_IMAGE_ORG}" "${PROBE_IMAGE_NAME}" "${PROBE_IMAGE_TAG}"
fi

load_external_config "cluster-${CLUSTER_NAME}" CLUSTER_
if [[ "${ENVIRONMENT}" != "dev" ]]; then
    oc login --token="${CLUSTER_ROBOT_OC_TOKEN}" --server="$CLUSTER_URL"
fi

NAMESPACE="rhacs-probe"
AUTH_TYPE="OCM"

invoke_helm "${SCRIPT_DIR}" rhacs-probe \
  --namespace "${NAMESPACE}" \
  --set authType="${AUTH_TYPE}" \
  --set clusterName="${CLUSTER_NAME}" \
  --set dataPlaneRegion="${CLUSTER_REGION}" \
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
