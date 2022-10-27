#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# shellcheck source=scripts/lib/external_config.sh
source "$SCRIPT_DIR/../../../scripts/lib/external_config.sh"

if [[ $# -ne 2 ]]; then
    echo "Usage: $0 [environment] [cluster]" >&2
    echo "Known environments: stage prod"
    echo "Cluster typically looks like: acs-{environment}-dp-01"
    echo ""
    echo "Note: you need to be logged into OCM for your environment's administrator"
    echo "Note: you need to be logged into OC for your cluster's administrator"
    exit 2
fi

ENVIRONMENT=$1
CLUSTER_NAME=$2

export AWS_PROFILE="$ENVIRONMENT"

# Loads config from the external storage to the environment and applying a prefix to a variable name (if exists).
load_external_config() {
    local service="$1"
    local prefix="${2:-}"
    eval "$(run_chamber env "$service" | sed -E "s/(^export +)(.*)/\1${prefix}\2/")"
}

init_chamber

load_external_config fleetshard-sync FLEETSHARD_SYNC_
load_external_config logging LOGGING_
load_external_config observability OBSERVABILITY_

case $ENVIRONMENT in
  stage)
    # TODO: Fetch OCM token and log in as appropriate user as part of script.
    EXPECT_OCM_ID="2ECw6PIE06TzjScQXe6QxMMt3Sa"

    FM_ENDPOINT="https://xtr6hh3mg6zc80v.api.stage.openshift.com"

    FLEETSHARD_SYNC_IMAGE="quay.io/app-sre/acs-fleet-manager:f6ad53e"

    OBSERVABILITY_GITHUB_TAG="master"
    OBSERVABILITY_OBSERVATORIUM_GATEWAY="https://observatorium-mst.api.stage.openshift.com"
    ;;

  prod)
    # TODO: Fetch OCM token and log in as appropriate user as part of script.
    EXPECT_OCM_ID="2BBslbGSQs5PS2HCfJKqOPcCN4r"

    FM_ENDPOINT="https://api.openshift.com"

    FLEETSHARD_SYNC_IMAGE="quay.io/app-sre/acs-fleet-manager:6da74ac"

    OBSERVABILITY_GITHUB_TAG="237418b6e0ac47499948fd00fb7bbcab63e535e6"  # pragma: allowlist secret
    OBSERVABILITY_OBSERVATORIUM_GATEWAY="https://observatorium-mst.api.openshift.com"
    ;;

  *)
    echo "Unknown environment ${ENVIRONMENT}"
    exit 2
    ;;
esac

OPERATOR_USE_UPSTREAM=false
OPERATOR_SOURCE="redhat-operators"

ACTUAL_OCM_ID=$(ocm whoami | jq -r '.id')
if [[ "${EXPECT_OCM_ID}" != "${ACTUAL_OCM_ID}" ]]; then
  echo "Must be logged into rhacs-managed-service-prod account in OCM to get cluster ID"
  exit 1
fi
CLUSTER_ID=$(ocm list cluster "${CLUSTER_NAME}" --no-headers --columns="ID")
if [[ -z "${CLUSTER_ID}" ]]; then
  echo "CLUSTER_ID is empty. Make sure ${CLUSTER_NAME} points to an existing cluster."
  exit 1
fi

## Uncomment this section if you want to deploy an upstream version of the operator.
## Update the global pull secret within the dataplane cluster to include the read-only credentials for quay.io/rhacs-eng
#QUAY_READ_ONLY_USERNAME=$(bw get username "66de0e1f-52fd-470b-ad9b-ae0701339dda")
#QUAY_READ_ONLY_PASSWORD=$(bw get password "66de0e1f-52fd-470b-ad9b-ae0701339dda")
#quay_basic_auth="${QUAY_READ_ONLY_USERNAME}:${QUAY_READ_ONLY_PASSWORD}"
#oc get secret/pull-secret -n openshift-config --template='{{index .data ".dockerconfigjson" | base64decode}}' > ./tmp-pull-secret.json
#oc registry login --registry="quay.io/rhacs-eng" --auth-basic="${quay_basic_auth}" --to=./tmp-pull-secret.json --skip-check
#oc set data secret/pull-secret -n openshift-config --from-file=.dockerconfigjson=./tmp-pull-secret.json
#rm ./tmp-pull-secret.json
#OPERATOR_USE_UPSTREAM=true
#OPERATOR_SOURCE="rhacs-operators"

# helm template ... to debug changes
helm upgrade rhacs-terraform "${SCRIPT_DIR}" \
  --install \
  --debug \
  --namespace rhacs \
  --create-namespace \
  --set acsOperator.enabled=true \
  --set acsOperator.source="${OPERATOR_SOURCE}" \
  --set acsOperator.sourceNamespace=openshift-marketplace \
  --set acsOperator.version=v3.72.0 \
  --set acsOperator.upstream="${OPERATOR_USE_UPSTREAM}" \
  --set fleetshardSync.image="${FLEETSHARD_SYNC_IMAGE}" \
  --set fleetshardSync.authType="RHSSO" \
  --set fleetshardSync.clusterId="${CLUSTER_ID}" \
  --set fleetshardSync.fleetManagerEndpoint="${FM_ENDPOINT}" \
  --set fleetshardSync.redHatSSO.clientId="${FLEETSHARD_SYNC_RHSSO_SERVICE_ACCOUNT_CLIENT_ID}" \
  --set fleetshardSync.redHatSSO.clientSecret="${FLEETSHARD_SYNC_RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET}" \
  --set logging.aws.accessKeyId="${LOGGING_AWS_ACCESS_KEY_ID}" \
  --set logging.aws.secretAccessKey="${LOGGING_AWS_SECRET_ACCESS_KEY}" \
  --set observability.github.accessToken="${OBSERVABILITY_GITHUB_ACCESS_TOKEN}" \
  --set observability.github.repository=https://api.github.com/repos/stackrox/rhacs-observability-resources/contents \
  --set observability.github.tag="${OBSERVABILITY_GITHUB_TAG}" \
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
