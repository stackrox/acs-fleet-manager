#!/usr/bin/env bash

#TODO(kovayur): enable and review all shellcheck exclusions (SC2034)
#set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# shellcheck source=scripts/lib/external_config.sh
source "$SCRIPT_DIR/../../scripts/lib/external_config.sh"


if [[ $# -ne 2 ]]; then
    echo "Usage: $0 [environment] [cluster]" >&2
    echo "Known environments: integration stage prod"
    echo "Cluster typically looks like: acs-{env}-dp-01"
    exit 2
fi

ENVIRONMENT=$1
CLUSTER_NAME=$2

export AWS_AUTH_HELPER="${AWS_AUTH_HELPER:-aws-saml}"

init_chamber

load_external_config fleetshard-sync FLEETSHARD_SYNC_
load_external_config cloudwatch-exporter CLOUDWATCH_EXPORTER_
load_external_config logging LOGGING_
load_external_config observability OBSERVABILITY_
load_external_config secured-cluster SECURED_CLUSTER_
load_external_config quay/rhacs-eng QUAY_

case $ENVIRONMENT in
  dev)
    FM_ENDPOINT="http://fleet-manager.rhacs.svc.cluster.local:8000"
    OBSERVABILITY_GITHUB_TAG="master"
    OBSERVABILITY_OBSERVATORIUM_GATEWAY="https://observatorium-mst.api.nonexistent.openshift.com"
    OBSERVABILITY_OPERATOR_VERSION="v4.2.1"
    OPERATOR_USE_UPSTREAM="false"
    # shellcheck disable=SC2034
    OPERATOR_CHANNEL="stable"
    # shellcheck disable=SC2034
    OPERATOR_VERSION="v4.2.0"
    FLEETSHARD_SYNC_CPU_REQUEST="${FLEETSHARD_SYNC_CPU_REQUEST:-"200m"}"
    FLEETSHARD_SYNC_MEMORY_REQUEST="${FLEETSHARD_SYNC_MEMORY_REQUEST:-"512Mi"}"
    FLEETSHARD_SYNC_CPU_LIMIT="${FLEETSHARD_SYNC_CPU_LIMIT:-"500m"}"
    FLEETSHARD_SYNC_MEMORY_LIMIT="${FLEETSHARD_SYNC_MEMORY_LIMIT:-"512Mi"}"
    SECURED_CLUSTER_ENABLED="false"
    ;;

  integration)
    FM_ENDPOINT="https://qj3layty4dynlnz.api.integration.openshift.com"
    OBSERVABILITY_GITHUB_TAG="master"
    OBSERVABILITY_OBSERVATORIUM_GATEWAY="https://observatorium-mst.api.stage.openshift.com"
    OBSERVABILITY_OPERATOR_VERSION="v4.2.1"
    OPERATOR_USE_UPSTREAM="false"
    # shellcheck disable=SC2034
    OPERATOR_CHANNEL="stable"
    # shellcheck disable=SC2034
    OPERATOR_VERSION="v4.2.0"
    FLEETSHARD_SYNC_CPU_REQUEST="${FLEETSHARD_SYNC_CPU_REQUEST:-"200m"}"
    FLEETSHARD_SYNC_MEMORY_REQUEST="${FLEETSHARD_SYNC_MEMORY_REQUEST:-"1024Mi"}"
    FLEETSHARD_SYNC_CPU_LIMIT="${FLEETSHARD_SYNC_CPU_LIMIT:-"1000m"}"
    FLEETSHARD_SYNC_MEMORY_LIMIT="${FLEETSHARD_SYNC_MEMORY_LIMIT:-"1024Mi"}"
    # shellcheck disable=SC2034
    SECURED_CLUSTER_ENABLED="false"  # TODO(ROX-18908): enable
    ;;

  stage)
    FM_ENDPOINT="https://xtr6hh3mg6zc80v.api.stage.openshift.com"
    OBSERVABILITY_GITHUB_TAG="stage"
    OBSERVABILITY_OBSERVATORIUM_GATEWAY="https://observatorium-mst.api.stage.openshift.com"
    OBSERVABILITY_OPERATOR_VERSION="v4.2.1"
    OPERATOR_USE_UPSTREAM="false"
    # shellcheck disable=SC2034
    OPERATOR_CHANNEL="stable"
    # shellcheck disable=SC2034
    OPERATOR_VERSION="v4.2.0"
    FLEETSHARD_SYNC_CPU_REQUEST="${FLEETSHARD_SYNC_CPU_REQUEST:-"200m"}"
    FLEETSHARD_SYNC_MEMORY_REQUEST="${FLEETSHARD_SYNC_MEMORY_REQUEST:-"1024Mi"}"
    FLEETSHARD_SYNC_CPU_LIMIT="${FLEETSHARD_SYNC_CPU_LIMIT:-"1000m"}"
    FLEETSHARD_SYNC_MEMORY_LIMIT="${FLEETSHARD_SYNC_MEMORY_LIMIT:-"1024Mi"}"
    # shellcheck disable=SC2034
    SECURED_CLUSTER_ENABLED="true"
    ;;

  prod)
    FM_ENDPOINT="https://api.openshift.com"
    OBSERVABILITY_GITHUB_TAG="production"
    OBSERVABILITY_OBSERVATORIUM_GATEWAY="https://observatorium-mst.api.openshift.com"
    OBSERVABILITY_OPERATOR_VERSION="v4.2.1"
    OPERATOR_USE_UPSTREAM="false"
    # shellcheck disable=SC2034
    OPERATOR_CHANNEL="stable"
    # shellcheck disable=SC2034
    OPERATOR_VERSION="v4.2.0"
    FLEETSHARD_SYNC_CPU_REQUEST="${FLEETSHARD_SYNC_CPU_REQUEST:-"200m"}"
    FLEETSHARD_SYNC_MEMORY_REQUEST="${FLEETSHARD_SYNC_MEMORY_REQUEST:-"1024Mi"}"
    FLEETSHARD_SYNC_CPU_LIMIT="${FLEETSHARD_SYNC_CPU_LIMIT:-"1000m"}"
    FLEETSHARD_SYNC_MEMORY_LIMIT="${FLEETSHARD_SYNC_MEMORY_LIMIT:-"1024Mi"}"
    # shellcheck disable=SC2034
    SECURED_CLUSTER_ENABLED="true"
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

FLEETSHARD_SYNC_ORG="app-sre"
FLEETSHARD_SYNC_IMAGE="acs-fleet-manager"
# Get HEAD for both main and production. This is the latest merged commit.
FLEETSHARD_SYNC_TAG="$(git rev-parse --short=7 HEAD)"

if [[ "${ADDON_DRY_RUN:-}" == "true" ]]; then
    "${SCRIPT_DIR}/../../scripts/check_image_exists.sh" "${FLEETSHARD_SYNC_ORG}" "${FLEETSHARD_SYNC_IMAGE}" "${FLEETSHARD_SYNC_TAG}" 0 || echo >&2 "Ignoring failed image check in dry-run mode."
else
    "${SCRIPT_DIR}/../../scripts/check_image_exists.sh" "${FLEETSHARD_SYNC_ORG}" "${FLEETSHARD_SYNC_IMAGE}" "${FLEETSHARD_SYNC_TAG}"
fi

echo "Loading external config: audit-logs/${CLUSTER_NAME}"
load_external_config "audit-logs/${CLUSTER_NAME}" AUDIT_LOGS_

echo "Loading external config: cluster-${CLUSTER_NAME}"
load_external_config "cluster-${CLUSTER_NAME}" CLUSTER_

OPERATOR_SOURCE="redhat-operators"
OPERATOR_USE_UPSTREAM="${OPERATOR_USE_UPSTREAM:-false}"
if [[ "${OPERATOR_USE_UPSTREAM}" == "true" ]]; then
    oc login --token="${CLUSTER_ROBOT_OC_TOKEN}" --server="$CLUSTER_URL"

    quay_basic_auth="${QUAY_READ_ONLY_USERNAME}:${QUAY_READ_ONLY_PASSWORD}"
    pull_secret_json="$(mktemp)"
    trap 'rm -f "${pull_secret_json}"' EXIT
    oc get secret/pull-secret -n openshift-config --template='{{index .data ".dockerconfigjson" | base64decode}}' > "${pull_secret_json}"
    oc registry login --registry="quay.io/rhacs-eng" --auth-basic="${quay_basic_auth}" --to="${pull_secret_json}" --skip-check
    oc set data secret/pull-secret -n openshift-config --from-file=.dockerconfigjson="${pull_secret_json}"
    # shellcheck disable=SC2034
    OPERATOR_SOURCE="rhacs-operators"
fi

ocm post "/api/clusters_mgmt/v1/clusters/${CLUSTER_ID}/addons" << EOF
{
    "addon": {
        "id": "acs-fleetshard"
    },
    "parameters": {
        "items": [
            { "id": "acscs-environment", "value": "${ENVIRONMENT}" },
            { "id": "cloudwatch-aws-access-key-id", "value": "${CLOUDWATCH_EXPORTER_AWS_ACCESS_KEY_ID:-}" },
            { "id": "cloudwatch-aws-secret-access-key", "value": "${CLOUDWATCH_EXPORTER_AWS_SECRET_ACCESS_KEY:-}" },
            { "id": "fleetshard-sync-auth-type", "value": "RHSSO" },
            { "id": "fleetshard-sync-aws-region", "value": "${CLUSTER_REGION}" },
            { "id": "fleetshard-sync-aws-role-arn", "value": "${FLEETSHARD_SYNC_AWS_ROLE_ARN}" },
            { "id": "fleetshard-sync-fleet-manager-endpoint", "value": "${FM_ENDPOINT}" },
            { "id": "fleetshard-sync-managed-db-enabled", "value": "true" },
            { "id": "fleetshard-sync-managed-db-performance-insights", "value": "true" },
            { "id": "fleetshard-sync-managed-db-security-group", "value": "${CLUSTER_MANAGED_DB_SECURITY_GROUP}" },
            { "id": "fleetshard-sync-managed-db-subnet-group", "value": "${CLUSTER_MANAGED_DB_SUBNET_GROUP}" },
            { "id": "fleetshard-sync-red-hat-sso-client-id", "value": "${FLEETSHARD_SYNC_RHSSO_SERVICE_ACCOUNT_CLIENT_ID}" },
            { "id": "fleetshard-sync-red-hat-sso-client-secret", "value": "${FLEETSHARD_SYNC_RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET}" },
            { "id": "fleetshard-sync-red-hat-sso-realm", "value": "redhat-external" },
            { "id": "fleetshard-sync-red-hat-sso-endpoint", "value": "https://sso.redhat.com" },
            { "id": "fleetshard-sync-telemetry-storage-endpoint", "value": "${FLEETSHARD_SYNC_TELEMETRY_STORAGE_ENDPOINT:-}" },
            { "id": "fleetshard-sync-telemetry-storage-key", "value": "${FLEETSHARD_SYNC_TELEMETRY_STORAGE_KEY:-}" },
            { "id": "fleetshard-sync-create-auth-provider", "value": "true" },
            { "id": "logging-aws-access-key-id", "value": "${LOGGING_AWS_ACCESS_KEY_ID}" },
            { "id": "logging-aws-secret-access-key", "value": "${LOGGING_AWS_SECRET_ACCESS_KEY}" },
            { "id": "logging-group-prefix", "value": "${CLUSTER_NAME}" },
            { "id": "logging-aws-region", "value": "us-east-1" },
            { "id": "observability-dead-mans-switch-url", "value": "${OBSERVABILITY_DEAD_MANS_SWITCH_URL}" },
            { "id": "observability-pagerduty-key", "value": "${OBSERVABILITY_PAGERDUTY_ROUTING_KEY}" },
            { "id": "observability-github-access-token", "value": "${OBSERVABILITY_GITHUB_ACCESS_TOKEN}" },
            { "id": "observability-github-repository", "value": "https://api.github.com/repos/stackrox/rhacs-observability-resources/contents" },
            { "id": "observability-github-tag", "value": "${OBSERVABILITY_GITHUB_TAG}" },
            { "id": "observability-operator-version", "value": "${OBSERVABILITY_OPERATOR_VERSION}" },
            { "id": "observability-observatorium-gateway", "value": "${OBSERVABILITY_OBSERVATORIUM_GATEWAY}" },
            { "id": "observability-observatorium-metrics-client-id", "value": "${OBSERVABILITY_OBSERVATORIUM_METRICS_CLIENT_ID}" },
            { "id": "observability-observatorium-metrics-secret", "value": "${OBSERVABILITY_OBSERVATORIUM_METRICS_SECRET}" },
            { "id": "observability-observatorium-auth-type", "value": "redhat" },
            { "id": "observability-observatorium-red-hat-sso-auth-server-url", "value": "https://sso.redhat.com/auth/" },
            { "id": "observability-observatorium-red-hat-sso-realm", "value": "redhat-external" }
        ]
    }
}
EOF
