#!/bin/bash
set -eo pipefail

# Requires: `jq`
# Requires: BitWarden CLI `bw`

if [[ $# -ne 2 ]]; then
    echo "Usage: $0 [environment] [cluster]" >&2
    echo "Known environments: stage"
    echo "Cluster typically looks like: acs-{environment}-dp-01"
    echo ""
    echo "Note: you will need to be logged in (oc login --token=...) to use Helm"
    exit 2
fi

ENVIRONMENT=$1
CLUSTER_NAME=$2

function ensure_bitwarden_session_exists () {
  # Check if we need to get a new BitWarden CLI Session Key.
  if [[ -z "$BW_SESSION" ]]; then
    if bw login --check; then
      # We don't have a session key but we are logged in, so unlock and store the session.
      export BW_SESSION=$(bw unlock --raw)
    else
      # We don't have a session key and are not logged in, so log in and store the session.
      export BW_SESSION=$(bw login --raw)
    fi
  fi
}

case $ENVIRONMENT in
  stage)
    EXPECT_OCM_ID="29ygxk0eRzRrhgQN96RdOYKt28e"
    ACTUAL_OCM_ID=$(ocm whoami | jq -r '.id')
    if [[ "${EXPECT_OCM_ID}" != "${ACTUAL_OCM_ID}" ]]; then
      echo "Must be logged into rhacs-managed-service account in OCM to get cluster ID"
      exit 1
    fi
    CLUSTER_ID=$(ocm list cluster "${CLUSTER_NAME}" --no-headers --columns="ID")

    FM_ENDPOINT="https://xtr6hh3mg6zc80v.api.stage.openshift.com"

    ensure_bitwarden_session_exists
    FLEETSHARD_SYNC_RED_HAT_SSO_CLIENT_ID=$(bw get username 028ce1a9-f751-4056-9c72-aea70052728b)
    FLEETSHARD_SYNC_RED_HAT_SSO_CLIENT_SECRET=$(bw get password 028ce1a9-f751-4056-9c72-aea70052728b)
    LOGGING_AWS_ACCESS_KEY_ID=$(bw get item "84e2d673-27dd-4e87-bb16-aee800da4d73" | jq '.fields[] | select(.name | contains("AccessKeyID")) | .value' --raw-output)
    LOGGING_AWS_SECRET_ACCESS_KEY=$(bw get item "84e2d673-27dd-4e87-bb16-aee800da4d73" | jq '.fields[] | select(.name | contains("SecretAccessKey")) | .value' --raw-output)
    OBSERVABILITY_GITHUB_ACCESS_TOKEN=$(bw get password eb7aecd3-b553-4999-b201-aebe01445822)
    OBSERVABILITY_OBSERVATORIUM_METRICS_CLIENT_ID="observatorium-rhacs-metrics-staging"
    OBSERVABILITY_OBSERVATORIUM_METRICS_SECRET=$(
        bw get item 510c8ed9-ba9f-46d9-b906-ae6100cf72f5 | \
        jq --arg OBSERVABILITY_OBSERVATORIUM_METRICS_CLIENT_ID "${OBSERVABILITY_OBSERVATORIUM_METRICS_CLIENT_ID}" \
            '.fields[] | select(.name | contains($OBSERVABILITY_OBSERVATORIUM_METRICS_CLIENT_ID)) | .value' --raw-output
    )
    ;;

  prod)
    echo "TODO: Handle environment 'prod'"
    exit 2
    ;;

  *)
    echo "Unknown environment ${ENVIRONMENT}"
    exit 2
    ;;
esac

# To use ACS Operator from the OpenShift Marketplace:
#   --set acsOperator.source=redhat-operators
#   --set acsOperator.sourceNamespace=openshift-marketplace

# helm template ... to debug changes
helm upgrade rhacs-terraform ./ \
  --install \
  --debug \
  --namespace rhacs \
  --create-namespace \
  --set acsOperator.enabled=true \
  --set acsOperator.source=rhacs-operators \
  --set acsOperator.startingCSV=rhacs-operator.v3.71.0 \
  --set fleetshardSync.authType="RHSSO" \
  --set fleetshardSync.clusterId=${CLUSTER_ID} \
  --set fleetshardSync.fleetManagerEndpoint=${FM_ENDPOINT} \
  --set fleetshardSync.redHatSSO.clientId="${FLEETSHARD_SYNC_RED_HAT_SSO_CLIENT_ID}" \
  --set fleetshardSync.redHatSSO.clientSecret="${FLEETSHARD_SYNC_RED_HAT_SSO_CLIENT_SECRET}" \
  --set logging.aws.accessKeyId="${LOGGING_AWS_ACCESS_KEY_ID}" \
  --set logging.aws.secretAccessKey="${LOGGING_AWS_SECRET_ACCESS_KEY}" \
  --set observability.enabled=true \
  --set observability.github.accessToken="${OBSERVABILITY_GITHUB_ACCESS_TOKEN}" \
  --set observability.github.repository=https://api.github.com/repos/stackrox/rhacs-observability-resources/contents \
  --set observability.gateway=https://observatorium-mst.api.stage.openshift.com \
  --set observability.observatorium.metricsClientId="${OBSERVABILITY_OBSERVATORIUM_METRICS_CLIENT_ID}" \
  --set observability.observatorium.metricsSecret="${OBSERVABILITY_OBSERVATORIUM_METRICS_SECRET}"

# To uninstall an existing release:
# helm uninstall rhacs-terraform --namespace rhacs
#
# To delete all resources specified by the template:
# helm template ... > /var/tmp/resources.yaml
# kubectl delete -f /var/tmp/resources.yaml
