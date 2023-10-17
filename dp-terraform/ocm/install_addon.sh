#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# shellcheck source=scripts/lib/external_config.sh
source "$SCRIPT_DIR/../../scripts/lib/external_config.sh"


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
    SECURED_CLUSTER_ENABLED="true"
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

OCM_COMMAND="patch"
OCM_ENDPOINT="/api/clusters_mgmt/v1/clusters/${CLUSTER_ID}/addons/acs-fleetshard"
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
            { "id": "cloudwatchAwsAccessKeyId", "value": "${CLOUDWATCH_EXPORTER_AWS_ACCESS_KEY_ID:-}" },
            { "id": "cloudwatchAwsSecretAccessKey", "value": "${CLOUDWATCH_EXPORTER_AWS_SECRET_ACCESS_KEY:-}" },
            { "id": "fleetshardSyncAuthType", "value": "RHSSO" },
            { "id": "fleetshardSyncImageTag", "value": "quay.io/${FLEETSHARD_SYNC_ORG}/${FLEETSHARD_SYNC_IMAGE}:${FLEETSHARD_SYNC_TAG}" },
            { "id": "fleetshardSyncAwsRegion", "value": "${CLUSTER_REGION}" },
            { "id": "fleetshardSyncAwsRoleArn", "value": "${FLEETSHARD_SYNC_AWS_ROLE_ARN}" },
            { "id": "fleetshardSyncFleetManagerEndpoint", "value": "${FM_ENDPOINT}" },
            { "id": "fleetshardSyncImageCredentialsPassword", "value": "${QUAY_READ_ONLY_PASSWORD}" },
            { "id": "fleetshardSyncImageCredentialsRegistry", "value": "quay.io" },
            { "id": "fleetshardSyncImageCredentialsUsername", "value": "${QUAY_READ_ONLY_USERNAME}" },
            { "id": "fleetshardSyncManagedDbEnabled", "value": "true" },
            { "id": "fleetshardSyncManagedDbPerformanceInsights", "value": "true" },
            { "id": "fleetshardSyncManagedDbSecurityGroup", "value": "${CLUSTER_MANAGED_DB_SECURITY_GROUP}" },
            { "id": "fleetshardSyncManagedDbSubnetGroup", "value": "${CLUSTER_MANAGED_DB_SUBNET_GROUP}" },
            { "id": "fleetshardSyncRedHatSsoClientId", "value": "${FLEETSHARD_SYNC_RHSSO_SERVICE_ACCOUNT_CLIENT_ID}" },
            { "id": "fleetshardSyncRedHatSsoClientSecret", "value": "${FLEETSHARD_SYNC_RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET}" },
            { "id": "fleetshardSyncRedHatSsoEndpoint", "value": "https://sso.redhat.com" },
            { "id": "fleetshardSyncRedHatSsoRealm", "value": "redhat-external" },
            { "id": "fleetshardSyncResourcesLimitsCpu", "value": "${FLEETSHARD_SYNC_CPU_LIMIT}" },
            { "id": "fleetshardSyncResourcesLimitsMemory", "value": "${FLEETSHARD_SYNC_MEMORY_LIMIT}" },
            { "id": "fleetshardSyncResourcesRequestsCpu", "value": "${FLEETSHARD_SYNC_CPU_REQUEST}" },
            { "id": "fleetshardSyncResourcesRequestsMemory", "value": "${FLEETSHARD_SYNC_MEMORY_REQUEST}" },
            { "id": "fleetshardSyncSecretEncryptionKeyId", "value": "${CLUSTER_SECRET_ENCRYPTION_KEY_ID}" },
            { "id": "fleetshardSyncSecretEncryptionType", "value": "kms" },
            { "id": "fleetshardSyncTelemetryStorageEndpoint", "value": "${FLEETSHARD_SYNC_TELEMETRY_STORAGE_ENDPOINT:-}" },
            { "id": "fleetshardSyncTelemetryStorageKey", "value": "${FLEETSHARD_SYNC_TELEMETRY_STORAGE_KEY:-}" },
            { "id": "loggingAwsAccessKeyId", "value": "${LOGGING_AWS_ACCESS_KEY_ID}" },
            { "id": "loggingAwsRegion", "value": "us-east-1" },
            { "id": "loggingAwsSecretAccessKey", "value": "${LOGGING_AWS_SECRET_ACCESS_KEY}" },
            { "id": "loggingGroupPrefix", "value": "${CLUSTER_NAME}" },
            { "id": "observabilityDeadMansSwitchUrl", "value": "${OBSERVABILITY_DEAD_MANS_SWITCH_URL}" },
            { "id": "observabilityGithubAccessToken", "value": "${OBSERVABILITY_GITHUB_ACCESS_TOKEN}" },
            { "id": "observabilityGithubRepository", "value": "https://api.github.com/repos/stackrox/rhacs-observability-resources/contents" },
            { "id": "observabilityGithubTag", "value": "${OBSERVABILITY_GITHUB_TAG}" },
            { "id": "observabilityObservatoriumAuthType", "value": "redhat" },
            { "id": "observabilityObservatoriumGateway", "value": "${OBSERVABILITY_OBSERVATORIUM_GATEWAY}" },
            { "id": "observabilityObservatoriumMetricsClientId", "value": "${OBSERVABILITY_OBSERVATORIUM_METRICS_CLIENT_ID}" },
            { "id": "observabilityObservatoriumMetricsSecret", "value": "${OBSERVABILITY_OBSERVATORIUM_METRICS_SECRET}" },
            { "id": "observabilityObservatoriumRedHatSsoAuthServerUrl", "value": "https://sso.redhat.com/auth/" },
            { "id": "observabilityObservatoriumRedHatSsoRealm", "value": "redhat-external" },
            { "id": "observabilityOperatorVersion", "value": "${OBSERVABILITY_OPERATOR_VERSION}" },
            { "id": "observabilityPagerdutyKey", "value": "${OBSERVABILITY_PAGERDUTY_ROUTING_KEY}" },
            { "id": "securedClusterAdmissionControlServiceTlsCert", "value": "${SECURED_CLUSTER_ADMISSION_CONTROL_CERT}" },
            { "id": "securedClusterAdmissionControlServiceTlsKey", "value": "${SECURED_CLUSTER_ADMISSION_CONTROL_KEY}" },
            { "id": "securedClusterCaCert", "value": "${SECURED_CLUSTER_CA_CERT}" },
            { "id": "securedClusterCentralEndpoint", "value": "${SECURED_CLUSTER_CENTRAL_ENDPOINT}" },
            { "id": "securedClusterCollectorServiceTlsCert", "value": "${SECURED_CLUSTER_COLLECTOR_CERT}" },
            { "id": "securedClusterCollectorServiceTlsKey", "value": "${SECURED_CLUSTER_COLLECTOR_KEY}" },
            { "id": "securedClusterEnabled", "value": "${SECURED_CLUSTER_ENABLED}" },
            { "id": "securedClusterSensorServiceTlsCert", "value": "${SECURED_CLUSTER_SENSOR_CERT}" },
            { "id": "securedClusterSensorServiceTlsKey", "value": "${SECURED_CLUSTER_SENSOR_KEY}" }
        ]
    }
}
EOF
)

# Check whether the addon is installed on a cluster
# If installed, using the idempotent patch command to update the parameters of the existing installation.
# Otherwise, use post endpoint to install.
if ! GET_ADDON_BODY=$(ocm get "/api/clusters_mgmt/v1/clusters/$CLUSTER_ID/addons/acs-fleetshard" 2>&1); then
    result=$(jq -r '.kind + ":" + .id' <<< "$GET_ADDON_BODY")
    if [[ "$result" != "Error:404" ]]; then
        echo 1>&2 "Unknown OCM error: $result"
        exit 1
    fi
    # Install the addon for the first time
    OCM_COMMAND="post"
    OCM_ENDPOINT="/api/clusters_mgmt/v1/clusters/${CLUSTER_ID}/addons"
    OCM_PAYLOAD=$(jq '. + {addon: { id: "acs-fleetshard" }}' <<< "$OCM_PAYLOAD")
fi

echo "Running 'ocm $OCM_COMMAND' to install the addon"

OCM_RESPONSE=$(ocm "$OCM_COMMAND" "$OCM_ENDPOINT" <<< "$OCM_PAYLOAD")

# Filtering sensitive fields
jq "{ kind, id, addon, addon_version, state, operator_version, csv_name, creation_timestamp, updated_timestamp }" <<< "$OCM_RESPONSE"
