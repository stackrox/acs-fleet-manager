#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# shellcheck source=scripts/lib/external_config.sh
source "$SCRIPT_DIR/../../../scripts/lib/external_config.sh"
# shellcheck source=scripts/lib/helm.sh
source "$SCRIPT_DIR/../../../scripts/lib/helm.sh"

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
    FM_ENDPOINT="https://nonexistent.api.stage.openshift.com"
    OBSERVABILITY_GITHUB_TAG="master"
    OBSERVABILITY_OBSERVATORIUM_GATEWAY="https://observatorium-mst.api.nonexistent.openshift.com"
    OBSERVABILITY_OPERATOR_VERSION="v4.2.1"
    OPERATOR_USE_UPSTREAM="false"
    OPERATOR_CHANNEL="stable"
    OPERATOR_VERSION="v4.0.2"
    FLEETSHARD_SYNC_CPU_REQUEST="${FLEETSHARD_SYNC_CPU_REQUEST:-"200m"}"
    FLEETSHARD_SYNC_MEMORY_REQUEST="${FLEETSHARD_SYNC_MEMORY_REQUEST:-"512Mi"}"
    FLEETSHARD_SYNC_CPU_LIMIT="${FLEETSHARD_SYNC_CPU_LIMIT:-"500m"}"
    FLEETSHARD_SYNC_MEMORY_LIMIT="${FLEETSHARD_SYNC_MEMORY_LIMIT:-"512Mi"}"
    SECURED_CLUSTER_ENABLED="true"
    ;;

  integration)
    FM_ENDPOINT="https://qj3layty4dynlnz.api.integration.openshift.com"
    OBSERVABILITY_GITHUB_TAG="master"
    OBSERVABILITY_OBSERVATORIUM_GATEWAY="https://observatorium-mst.api.stage.openshift.com"
    OBSERVABILITY_OPERATOR_VERSION="v4.2.1"
    OPERATOR_USE_UPSTREAM="false"
    OPERATOR_CHANNEL="stable"
    OPERATOR_VERSION="v4.1.0"
    FLEETSHARD_SYNC_CPU_REQUEST="${FLEETSHARD_SYNC_CPU_REQUEST:-"200m"}"
    FLEETSHARD_SYNC_MEMORY_REQUEST="${FLEETSHARD_SYNC_MEMORY_REQUEST:-"1024Mi"}"
    FLEETSHARD_SYNC_CPU_LIMIT="${FLEETSHARD_SYNC_CPU_LIMIT:-"1000m"}"
    FLEETSHARD_SYNC_MEMORY_LIMIT="${FLEETSHARD_SYNC_MEMORY_LIMIT:-"1024Mi"}"
    SECURED_CLUSTER_ENABLED="false"  # TODO(ROX-18908): enable
    ;;

  stage)
    FM_ENDPOINT="https://xtr6hh3mg6zc80v.api.stage.openshift.com"
    OBSERVABILITY_GITHUB_TAG="master"
    OBSERVABILITY_OBSERVATORIUM_GATEWAY="https://observatorium-mst.api.stage.openshift.com"
    OBSERVABILITY_OPERATOR_VERSION="v4.2.1"
    OPERATOR_USE_UPSTREAM="false"
    OPERATOR_CHANNEL="stable"
    OPERATOR_VERSION="v4.1.0"
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
    OPERATOR_USE_UPSTREAM="false"
    OPERATOR_CHANNEL="stable"
    OPERATOR_VERSION="v4.1.0"
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

if [[ "${HELM_DRY_RUN:-}" == "true" ]]; then
    "${SCRIPT_DIR}/../../../scripts/check_image_exists.sh" "${FLEETSHARD_SYNC_ORG}" "${FLEETSHARD_SYNC_IMAGE}" "${FLEETSHARD_SYNC_TAG}" 0 || echo >&2 "Ignoring failed image check in dry-run mode."
else
    "${SCRIPT_DIR}/../../../scripts/check_image_exists.sh" "${FLEETSHARD_SYNC_ORG}" "${FLEETSHARD_SYNC_IMAGE}" "${FLEETSHARD_SYNC_TAG}"
fi

echo "Loading external config: audit-logs/${CLUSTER_NAME}"
load_external_config "audit-logs/${CLUSTER_NAME}" AUDIT_LOGS_

echo "Loading external config: cluster-${CLUSTER_NAME}"
load_external_config "cluster-${CLUSTER_NAME}" CLUSTER_
if [[ "${ENVIRONMENT}" != "dev" ]]; then
    oc login --token="${CLUSTER_ROBOT_OC_TOKEN}" --server="$CLUSTER_URL"
fi

OPERATOR_SOURCE="redhat-operators"
OPERATOR_USE_UPSTREAM="${OPERATOR_USE_UPSTREAM:-false}"
if [[ "${OPERATOR_USE_UPSTREAM}" == "true" ]]; then
    quay_basic_auth="${QUAY_READ_ONLY_USERNAME}:${QUAY_READ_ONLY_PASSWORD}"
    pull_secret_json="$(mktemp)"
    trap 'rm -f "${pull_secret_json}"' EXIT
    oc get secret/pull-secret -n openshift-config --template='{{index .data ".dockerconfigjson" | base64decode}}' > "${pull_secret_json}"
    oc registry login --registry="quay.io/rhacs-eng" --auth-basic="${quay_basic_auth}" --to="${pull_secret_json}" --skip-check
    oc set data secret/pull-secret -n openshift-config --from-file=.dockerconfigjson="${pull_secret_json}"

    OPERATOR_SOURCE="rhacs-operators"
fi

# TODO(ROX-16771): Move this to env-specific values.yaml files
# TODO(ROX-16645): set acsOperator.enabled to false
invoke_helm "${SCRIPT_DIR}" rhacs-terraform \
  --namespace rhacs \
  --set acsOperator.enabled=true \
  --set acsOperator.source="${OPERATOR_SOURCE}" \
  --set acsOperator.sourceNamespace=openshift-marketplace \
  --set acsOperator.channel="${OPERATOR_CHANNEL}" \
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
  --set fleetshardSync.aws.region="${CLUSTER_REGION}" \
  --set fleetshardSync.aws.roleARN="${FLEETSHARD_SYNC_AWS_ROLE_ARN}" \
  --set fleetshardSync.telemetry.storage.endpoint="${FLEETSHARD_SYNC_TELEMETRY_STORAGE_ENDPOINT:-}" \
  --set fleetshardSync.telemetry.storage.key="${FLEETSHARD_SYNC_TELEMETRY_STORAGE_KEY:-}" \
  --set fleetshardSync.resources.requests.cpu="${FLEETSHARD_SYNC_CPU_REQUEST}" \
  --set fleetshardSync.resources.requests.memory="${FLEETSHARD_SYNC_MEMORY_REQUEST}" \
  --set fleetshardSync.resources.limits.cpu="${FLEETSHARD_SYNC_CPU_LIMIT}" \
  --set fleetshardSync.resources.limits.memory="${FLEETSHARD_SYNC_MEMORY_LIMIT}" \
  --set fleetshardSync.imageCredentials.registry="quay.io" \
  --set fleetshardSync.imageCredentials.username="${QUAY_READ_ONLY_USERNAME}" \
  --set fleetshardSync.imageCredentials.password="${QUAY_READ_ONLY_PASSWORD}" \
  --set fleetshardSync.secretEncryption.type="kms" \
  --set fleetshardSync.secretEncryption.keyID="${CLUSTER_SECRET_ENCRYPTION_KEY_ID}" \
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
  --set observability.pagerduty.key="${OBSERVABILITY_PAGERDUTY_ROUTING_KEY}" \
  --set observability.deadMansSwitch.url="${OBSERVABILITY_DEAD_MANS_SWITCH_URL}" \
  --set audit-logs.enabled=true \
  --set audit-logs.annotations.rhacs\\.redhat\\.com/cluster-name="${CLUSTER_NAME}" \
  --set audit-logs.annotations.rhacs\\.redhat\\.com/environment="${ENVIRONMENT}" \
  --set audit-logs.customConfig.sinks.aws_cloudwatch_logs.group_name="${AUDIT_LOGS_LOG_GROUP_NAME}" \
  --set audit-logs.secrets.aws_role_arn="${AUDIT_LOGS_ROLE_ARN:-}" \
  --set secured-cluster.enabled="${SECURED_CLUSTER_ENABLED}" \
  --set secured-cluster.clusterName="${CLUSTER_NAME}" \
  --set secured-cluster.centralEndpoint="${SECURED_CLUSTER_CENTRAL_ENDPOINT}" \
  --set secured-cluster.ca.cert="${SECURED_CLUSTER_CA_CERT}" \
  --set secured-cluster.admissionControl.serviceTLS.cert="${SECURED_CLUSTER_ADMISSION_CONTROL_CERT}" \
  --set secured-cluster.admissionControl.serviceTLS.key="${SECURED_CLUSTER_ADMISSION_CONTROL_KEY}" \
  --set secured-cluster.collector.serviceTLS.cert="${SECURED_CLUSTER_COLLECTOR_CERT}" \
  --set secured-cluster.collector.serviceTLS.key="${SECURED_CLUSTER_COLLECTOR_KEY}" \
  --set secured-cluster.sensor.serviceTLS.cert="${SECURED_CLUSTER_SENSOR_CERT}" \
  --set secured-cluster.sensor.serviceTLS.key="${SECURED_CLUSTER_SENSOR_KEY}"

# To uninstall an existing release:
# helm uninstall rhacs-terraform --namespace rhacs
#
# To delete all resources specified by the template:
# helm template ... > /var/tmp/resources.yaml
# kubectl delete -f /var/tmp/resources.yaml
