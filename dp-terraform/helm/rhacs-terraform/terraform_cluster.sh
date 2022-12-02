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
load_external_config logging LOGGING_
load_external_config observability OBSERVABILITY_

case $ENVIRONMENT in
  stage)
    FM_ENDPOINT="https://xtr6hh3mg6zc80v.api.stage.openshift.com"
    OBSERVABILITY_GITHUB_TAG="master"
    OBSERVABILITY_OBSERVATORIUM_GATEWAY="https://observatorium-mst.api.stage.openshift.com"
    ;;

  prod)
    FM_ENDPOINT="https://api.openshift.com"
    OBSERVABILITY_GITHUB_TAG="production"
    OBSERVABILITY_OBSERVATORIUM_GATEWAY="https://observatorium-mst.api.openshift.com"
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

# Get the first non-merge commit, starting with HEAD.
# On main this should be HEAD, on production, the latest merged main commit.
FLEETSHARD_SYNC_TAG="$(git rev-list --no-merges --max-count 1 --abbrev-commit --abbrev=7 HEAD)"
"${SCRIPT_DIR}/check_image_exists.sh" "${FLEETSHARD_SYNC_TAG}"

load_external_config "cluster-${CLUSTER_NAME}" CLUSTER_
oc login --token="${CLUSTER_ROBOT_OC_TOKEN}" --server="$CLUSTER_URL"

OPERATOR_USE_UPSTREAM=false
OPERATOR_SOURCE="redhat-operators"

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

# helm template --debug ... to debug changes
helm upgrade rhacs-terraform "${SCRIPT_DIR}" \
  --install \
  --namespace rhacs \
  --create-namespace \
  --set acsOperator.enabled=true \
  --set acsOperator.source="${OPERATOR_SOURCE}" \
  --set acsOperator.sourceNamespace=openshift-marketplace \
  --set acsOperator.version=v3.72.0 \
  --set acsOperator.upstream="${OPERATOR_USE_UPSTREAM}" \
  --set fleetshardSync.image="quay.io/app-sre/acs-fleet-manager:${FLEETSHARD_SYNC_TAG}" \
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
