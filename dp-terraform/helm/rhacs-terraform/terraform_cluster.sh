#!/bin/bash
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

export AWS_AUTH_HELPER="${AWS_AUTH_HELPER:-aws-saml}"
if [[ "$AWS_AUTH_HELPER" == "aws-vault" ]]; then
    export AWS_PROFILE="$ENVIRONMENT"
fi

init_chamber

load_external_config fleetshard-sync FLEETSHARD_SYNC_
load_external_config cloudwatch-exporter CLOUDWATCH_EXPORTER_
load_external_config logging LOGGING_
load_external_config observability OBSERVABILITY_

case $ENVIRONMENT in
  stage)
    FM_ENDPOINT="https://xtr6hh3mg6zc80v.api.stage.openshift.com"
    OBSERVABILITY_GITHUB_TAG="master"
    OBSERVABILITY_OBSERVATORIUM_GATEWAY="https://observatorium-mst.api.stage.openshift.com"
    OBSERVABILITY_OPERATOR_VERSION="v4.0.4"
    OPERATOR_USE_UPSTREAM="false"
    OPERATOR_VERSION="v3.74.0"
    ;;

  prod)
    FM_ENDPOINT="https://api.openshift.com"
    OBSERVABILITY_GITHUB_TAG="production"
    OBSERVABILITY_OBSERVATORIUM_GATEWAY="https://observatorium-mst.api.openshift.com"
    OBSERVABILITY_OPERATOR_VERSION="v4.0.4"
    OPERATOR_USE_UPSTREAM="false"
    OPERATOR_VERSION="v3.74.0"
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

FLEETSHARD_SYNC_ORG="app-sre"
FLEETSHARD_SYNC_IMAGE="acs-fleet-manager"
# Get the first non-merge commit, starting with HEAD.
# On main this should be HEAD, on production, the latest merged main commit.
FLEETSHARD_SYNC_TAG="$(git rev-list --no-merges --max-count 1 --abbrev-commit --abbrev=7 HEAD)"

if [[ "${HELM_PRINT_ONLY:-}" == "true" ]]; then
    HELM_DEBUG_FLAGS="--debug --dry-run"
else
    "${SCRIPT_DIR}/../../../scripts/check_image_exists.sh" "${FLEETSHARD_SYNC_ORG}" "${FLEETSHARD_SYNC_IMAGE}" "${FLEETSHARD_SYNC_TAG}"
fi

load_external_config "cluster-${CLUSTER_NAME}" CLUSTER_
oc login --token="${CLUSTER_ROBOT_OC_TOKEN}" --server="$CLUSTER_URL"

OPERATOR_SOURCE="redhat-operators"
OPERATOR_USE_UPSTREAM="${OPERATOR_USE_UPSTREAM:-false}"
if [[ "${OPERATOR_USE_UPSTREAM}" == "true" ]]; then
    load_external_config quay/rhacs-eng QUAY_
    quay_basic_auth="${QUAY_READ_ONLY_USERNAME}:${QUAY_READ_ONLY_PASSWORD}"
    pull_secret_json="$(mktemp)"
    trap 'rm -f "${pull_secret_json}"' EXIT
    oc get secret/pull-secret -n openshift-config --template='{{index .data ".dockerconfigjson" | base64decode}}' > "${pull_secret_json}"
    oc registry login --registry="quay.io/rhacs-eng" --auth-basic="${quay_basic_auth}" --to="${pull_secret_json}" --skip-check
    oc set data secret/pull-secret -n openshift-config --from-file=.dockerconfigjson="${pull_secret_json}"

    OPERATOR_SOURCE="rhacs-operators"
fi

# shellcheck disable=SC2086
helm upgrade rhacs-terraform "${SCRIPT_DIR}" ${HELM_DEBUG_FLAGS:-} \
  --install \
  --namespace rhacs \
  --create-namespace \
  --set acsOperator.enabled=true \
  --set acsOperator.source="${OPERATOR_SOURCE}" \
  --set acsOperator.sourceNamespace=openshift-marketplace \
  --set acsOperator.version="${OPERATOR_VERSION}" \
  --set acsOperator.upstream="${OPERATOR_USE_UPSTREAM}" \
  --set fleetshardSync.image="quay.io/${FLEETSHARD_SYNC_ORG}/${FLEETSHARD_SYNC_IMAGE}:${FLEETSHARD_SYNC_TAG}" \
  --set fleetshardSync.authType="RHSSO" \
  --set fleetshardSync.clusterId="${CLUSTER_ID}" \
  --set fleetshardSync.clusterName="${CLUSTER_NAME}" \
  --set fleetshardSync.environment="${ENVIRONMENT}" \
  --set fleetshardSync.fleetManagerEndpoint="${FM_ENDPOINT}" \
  --set fleetshardSync.redHatSSO.clientId="${FLEETSHARD_SYNC_RHSSO_SERVICE_ACCOUNT_CLIENT_ID}" \
  --set fleetshardSync.redHatSSO.clientSecret="${FLEETSHARD_SYNC_RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET}" \
  --set fleetshardSync.managedDB.enabled=true \
  --set fleetshardSync.managedDB.subnetGroup="${CLUSTER_MANAGED_DB_SUBNET_GROUP}" \
  --set fleetshardSync.managedDB.securityGroup="${CLUSTER_MANAGED_DB_SECURITY_GROUP}" \
  --set fleetshardSync.managedDB.performanceInsights=true \
  --set fleetshardSync.aws.region="${FLEETSHARD_SYNC_AWS_REGION}" \
  --set fleetshardSync.aws.roleARN="${FLEETSHARD_SYNC_AWS_ROLE_ARN}" \
  --set fleetshardSync.telemetry.storage.endpoint="${FLEETSHARD_SYNC_TELEMETRY_STORAGE_ENDPOINT:-}" \
  --set fleetshardSync.telemetry.storage.key="${FLEETSHARD_SYNC_TELEMETRY_STORAGE_KEY:-}" \
  --set cloudwatch.aws.accessKeyId="${CLOUDWATCH_EXPORTER_AWS_ACCESS_KEY_ID:-}" \
  --set cloudwatch.aws.secretAccessKey="${CLOUDWATCH_EXPORTER_AWS_SECRET_ACCESS_KEY:-}" \
  --set cloudwatch.clusterName="${CLUSTER_NAME}" \
  --set cloudwatch.environment="${ENVIRONMENT}" \
  --set logging.groupPrefix="${CLUSTER_NAME}" \
  --set logging.aws.accessKeyId="${LOGGING_AWS_ACCESS_KEY_ID}" \
  --set logging.aws.secretAccessKey="${LOGGING_AWS_SECRET_ACCESS_KEY}" \
  --set observability.clusterName="${CLUSTER_NAME}" \
  --set observability.github.accessToken="${OBSERVABILITY_GITHUB_ACCESS_TOKEN}" \
  --set observability.github.repository=https://api.github.com/repos/stackrox/rhacs-observability-resources/contents \
  --set observability.github.tag="${OBSERVABILITY_GITHUB_TAG}" \
  --set observability.observabilityOperatorVersion="${OBSERVABILITY_OPERATOR_VERSION}" \
  --set observability.observatorium.gateway="${OBSERVABILITY_OBSERVATORIUM_GATEWAY}" \
  --set observability.observatorium.metricsClientId="${OBSERVABILITY_OBSERVATORIUM_METRICS_CLIENT_ID}" \
  --set observability.observatorium.metricsSecret="${OBSERVABILITY_OBSERVATORIUM_METRICS_SECRET}" \
  --set observability.pagerduty.key="${OBSERVABILITY_PAGERDUTY_SERVICE_KEY}" \
  --set observability.deadMansSwitch.url="${OBSERVABILITY_DEAD_MANS_SWITCH_URL}"

# To uninstall an existing release:
# helm uninstall rhacs-terraform --namespace rhacs
#
# To delete all resources specified by the template:
# helm template ... > /var/tmp/resources.yaml
# kubectl delete -f /var/tmp/resources.yaml
