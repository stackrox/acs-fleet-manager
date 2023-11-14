#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$SCRIPT_DIR/../.."

# shellcheck source=scripts/lib/external_config.sh
source "$ROOT_DIR/scripts/lib/external_config.sh"

if [[ $# -ne 2 ]]; then
    echo "Usage: $0 [environment] [cluster]" >&2
    echo "Known environments: dev integration stage prod"
    echo "Cluster typically looks like: acs-{env}-dp-01"
    exit 2
fi

ENVIRONMENT=$1
CLUSTER_NAME=$2

export AWS_AUTH_HELPER="${AWS_AUTH_HELPER:-aws-saml}"

init_chamber

load_external_config secured-cluster SECURED_CLUSTER_

case $ENVIRONMENT in
  dev)
    FM_ENDPOINT="http://fleet-manager.rhacs.svc.cluster.local:8000"
    OBSERVABILITY_GITHUB_TAG="master"
    OBSERVABILITY_OBSERVATORIUM_GATEWAY="https://observatorium-mst.api.nonexistent.openshift.com"
    OBSERVABILITY_OPERATOR_VERSION="v4.2.1"
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
    FLEETSHARD_SYNC_CPU_REQUEST="${FLEETSHARD_SYNC_CPU_REQUEST:-"200m"}"
    FLEETSHARD_SYNC_MEMORY_REQUEST="${FLEETSHARD_SYNC_MEMORY_REQUEST:-"1024Mi"}"
    FLEETSHARD_SYNC_CPU_LIMIT="${FLEETSHARD_SYNC_CPU_LIMIT:-"1000m"}"
    FLEETSHARD_SYNC_MEMORY_LIMIT="${FLEETSHARD_SYNC_MEMORY_LIMIT:-"1024Mi"}"
    SECURED_CLUSTER_ENABLED="false"  # TODO(ROX-18908): enable
    ;;

  stage)
    FM_ENDPOINT="https://xtr6hh3mg6zc80v.api.stage.openshift.com"
    OBSERVABILITY_GITHUB_TAG="stage"
    OBSERVABILITY_OBSERVATORIUM_GATEWAY="https://observatorium-mst.api.stage.openshift.com"
    OBSERVABILITY_OPERATOR_VERSION="v4.2.1"
    FLEETSHARD_SYNC_CPU_REQUEST="${FLEETSHARD_SYNC_CPU_REQUEST:-"200m"}"
    FLEETSHARD_SYNC_MEMORY_REQUEST="${FLEETSHARD_SYNC_MEMORY_REQUEST:-"1024Mi"}"
    FLEETSHARD_SYNC_CPU_LIMIT="${FLEETSHARD_SYNC_CPU_LIMIT:-"1000m"}"
    FLEETSHARD_SYNC_MEMORY_LIMIT="${FLEETSHARD_SYNC_MEMORY_LIMIT:-"1024Mi"}"
    SECURED_CLUSTER_ENABLED="true"
    ;;

  prod)
    FM_ENDPOINT="https://api.openshift.com"
    OBSERVABILITY_GITHUB_TAG="production"
    OBSERVABILITY_OBSERVATORIUM_GATEWAY="https://observatorium-mst.api.openshift.com"
    OBSERVABILITY_OPERATOR_VERSION="v4.2.1"
    FLEETSHARD_SYNC_CPU_REQUEST="${FLEETSHARD_SYNC_CPU_REQUEST:-"200m"}"
    FLEETSHARD_SYNC_MEMORY_REQUEST="${FLEETSHARD_SYNC_MEMORY_REQUEST:-"1024Mi"}"
    FLEETSHARD_SYNC_CPU_LIMIT="${FLEETSHARD_SYNC_CPU_LIMIT:-"1000m"}"
    FLEETSHARD_SYNC_MEMORY_LIMIT="${FLEETSHARD_SYNC_MEMORY_LIMIT:-"1024Mi"}"
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
FLEETSHARD_SYNC_TAG="$(make --quiet --no-print-directory -C "${ROOT_DIR}" tag)"

if [[ "${ADDON_DRY_RUN:-}" == "true" ]]; then
    "${ROOT_DIR}/scripts/check_image_exists.sh" "${FLEETSHARD_SYNC_ORG}" "${FLEETSHARD_SYNC_IMAGE}" "${FLEETSHARD_SYNC_TAG}" 0 || echo >&2 "Ignoring failed image check in dry-run mode."
else
    "${ROOT_DIR}/scripts/check_image_exists.sh" "${FLEETSHARD_SYNC_ORG}" "${FLEETSHARD_SYNC_IMAGE}" "${FLEETSHARD_SYNC_TAG}"
fi

echo "Loading external config: audit-logs/${CLUSTER_NAME}"
load_external_config "audit-logs/${CLUSTER_NAME}" AUDIT_LOGS_

echo "Loading external config: cluster-${CLUSTER_NAME}"
load_external_config "cluster-${CLUSTER_NAME}" CLUSTER_

# Replace all the line breaks with \n
escape_linebreaks() {
    <<<"$1" sed '$ ! s/$/\\n/' | tr -d '\n'
}

# Allows to load an external cluster config (e.g. acs-dev-dp-01) and apply it to a different cluster with override
OCM_CLUSTER_ID="${OVERRIDE_CLUSTER_ID:-${CLUSTER_ID}}"

OCM_SUBSCRIPTION_ID=$(ocm get cluster "$OCM_CLUSTER_ID" | jq -r '.subscription.id')
AWS_ACCOUNT_ID=$(ocm get "/api/accounts_mgmt/v1/subscriptions/${OCM_SUBSCRIPTION_ID}" | jq -r '.cloud_account_id')

OCM_COMMAND="patch"
OCM_ENDPOINT="/api/clusters_mgmt/v1/clusters/${OCM_CLUSTER_ID}/addons/acs-fleetshard"
OCM_PAYLOAD=$(cat << EOF
{
    "addon_version": {
        "id": "0.2.0"
    },
    "parameters": {
        "items": [
            { "id": "acscsEnvironment", "value": "${ENVIRONMENT}" },
            { "id": "auditLogsLogGroupName", "value": "${AUDIT_LOGS_LOG_GROUP_NAME:-}" },
            { "id": "auditLogsRoleArn", "value": "${AUDIT_LOGS_ROLE_ARN:-}" },
            { "id": "fleetshardSyncAuthType", "value": "RHSSO" },
            { "id": "fleetshardSyncImageTag", "value": "quay.io/${FLEETSHARD_SYNC_ORG}/${FLEETSHARD_SYNC_IMAGE}:${FLEETSHARD_SYNC_TAG}" },
            { "id": "fleetshardSyncAwsRegion", "value": "${CLUSTER_REGION}" },
            { "id": "fleetshardSyncFleetManagerEndpoint", "value": "${FM_ENDPOINT}" },
            { "id": "fleetshardSyncManagedDbEnabled", "value": "true" },
            { "id": "fleetshardSyncManagedDbPerformanceInsights", "value": "true" },
            { "id": "fleetshardSyncManagedDbSecurityGroup", "value": "${CLUSTER_MANAGED_DB_SECURITY_GROUP}" },
            { "id": "fleetshardSyncManagedDbSubnetGroup", "value": "${CLUSTER_MANAGED_DB_SUBNET_GROUP}" },
            { "id": "fleetshardSyncRedHatSsoEndpoint", "value": "https://sso.redhat.com" },
            { "id": "fleetshardSyncRedHatSsoRealm", "value": "redhat-external" },
            { "id": "fleetshardSyncResourcesLimitsCpu", "value": "${FLEETSHARD_SYNC_CPU_LIMIT}" },
            { "id": "fleetshardSyncResourcesLimitsMemory", "value": "${FLEETSHARD_SYNC_MEMORY_LIMIT}" },
            { "id": "fleetshardSyncResourcesRequestsCpu", "value": "${FLEETSHARD_SYNC_CPU_REQUEST}" },
            { "id": "fleetshardSyncResourcesRequestsMemory", "value": "${FLEETSHARD_SYNC_MEMORY_REQUEST}" },
            { "id": "fleetshardSyncSecretEncryptionKeyId", "value": "${CLUSTER_SECRET_ENCRYPTION_KEY_ID}" },
            { "id": "fleetshardSyncSecretEncryptionType", "value": "kms" },
            { "id": "loggingAwsRegion", "value": "us-east-1" },
            { "id": "loggingGroupPrefix", "value": "${CLUSTER_NAME}" },
            { "id": "observabilityGithubTag", "value": "${OBSERVABILITY_GITHUB_TAG}" },
            { "id": "observabilityObservatoriumAuthType", "value": "redhat" },
            { "id": "observabilityObservatoriumGateway", "value": "${OBSERVABILITY_OBSERVATORIUM_GATEWAY}" },
            { "id": "observabilityOperatorVersion", "value": "${OBSERVABILITY_OPERATOR_VERSION}" },
            { "id": "securedClusterCentralEndpoint", "value": "${SECURED_CLUSTER_CENTRAL_ENDPOINT}" },
            { "id": "securedClusterEnabled", "value": "${SECURED_CLUSTER_ENABLED}" },
            { "id": "externalSecretsAwsRoleArn", "value": "arn:aws:iam::${AWS_ACCOUNT_ID}:role/ExternalSecretsServiceRole" }
        ]
    }
}
EOF
)

# Check whether the addon is installed on a cluster
# If installed, using the idempotent patch command to update the parameters of the existing installation.
# Otherwise, use post endpoint to install.
if ! GET_ADDON_BODY=$(ocm get "/api/clusters_mgmt/v1/clusters/$OCM_CLUSTER_ID/addons/acs-fleetshard" 2>&1); then
    result=$(jq -r '.kind + ":" + .id' <<< "$GET_ADDON_BODY")
    if [[ "$result" != "Error:404" ]]; then
        echo 1>&2 "Unknown OCM error: $result"
        exit 1
    fi
    # Install the addon for the first time
    OCM_COMMAND="post"
    OCM_ENDPOINT="/api/clusters_mgmt/v1/clusters/${OCM_CLUSTER_ID}/addons"
    OCM_PAYLOAD=$(jq '. + {addon: { id: "acs-fleetshard" }}' <<< "$OCM_PAYLOAD")
fi

echo "Running 'ocm $OCM_COMMAND' to install the addon"

OCM_RESPONSE=$(ocm "$OCM_COMMAND" "$OCM_ENDPOINT" <<< "$OCM_PAYLOAD")

# Filtering sensitive fields
jq "{ kind, id, addon, addon_version, state, operator_version, csv_name, creation_timestamp, updated_timestamp }" <<< "$OCM_RESPONSE"
