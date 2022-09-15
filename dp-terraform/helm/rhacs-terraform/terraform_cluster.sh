#!/bin/bash
set -eo pipefail

# Requires: `jq`
# Requires: BitWarden CLI `bw`

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
    # TODO: Fetch OCM token and log in as appropriate user as part of script.
    EXPECT_OCM_ID="2ECw6PIE06TzjScQXe6QxMMt3Sa"
    ACTUAL_OCM_ID=$(ocm whoami | jq -r '.id')
    if [[ "${EXPECT_OCM_ID}" != "${ACTUAL_OCM_ID}" ]]; then
      echo "Must be logged into rhacs-managed-service-stage account in OCM to get cluster ID"
      exit 1
    fi
    CLUSTER_ID=$(ocm list cluster "${CLUSTER_NAME}" --no-headers --columns="ID")

    FM_ENDPOINT="https://xtr6hh3mg6zc80v.api.stage.openshift.com"

    ensure_bitwarden_session_exists
    # Note: the Red Hat SSO client as of 2022-09-02 is the same between stage and prod.
    FLEETSHARD_SYNC_RED_HAT_SSO_CLIENT_ID=$(bw get username 028ce1a9-f751-4056-9c72-aea70052728b)
    FLEETSHARD_SYNC_RED_HAT_SSO_CLIENT_SECRET=$(bw get password 028ce1a9-f751-4056-9c72-aea70052728b)
    LOGGING_AWS_ACCESS_KEY_ID=$(bw get item "84e2d673-27dd-4e87-bb16-aee800da4d73" | jq '.fields[] | select(.name == "AccessKeyID") | .value' --raw-output)
    LOGGING_AWS_SECRET_ACCESS_KEY=$(bw get item "84e2d673-27dd-4e87-bb16-aee800da4d73" | jq '.fields[] | select(.name == "SecretAccessKey") | .value' --raw-output)
    # Note: the GitHub Access Token as of 2022-09-02 is the same between stage and prod.
    OBSERVABILITY_GITHUB_ACCESS_TOKEN=$(bw get password eb7aecd3-b553-4999-b201-aebe01445822)
    OBSERVABILITY_OBSERVATORIUM_GATEWAY="https://observatorium-mst.api.stage.openshift.com"
    OBSERVABILITY_OBSERVATORIUM_METRICS_CLIENT_ID="observatorium-rhacs-metrics-staging"
    OBSERVABILITY_OBSERVATORIUM_METRICS_SECRET=$(
        bw get item 510c8ed9-ba9f-46d9-b906-ae6100cf72f5 | \
        jq --arg OBSERVABILITY_OBSERVATORIUM_METRICS_CLIENT_ID "${OBSERVABILITY_OBSERVATORIUM_METRICS_CLIENT_ID}" \
            '.fields[] | select(.name == $OBSERVABILITY_OBSERVATORIUM_METRICS_CLIENT_ID) | .value' --raw-output
    )
    # Note: the PagerDuty Service Key as of 2022-09-02 is the same between stage and prod.
    PAGERDUTY_SERVICE_KEY=$(bw get item "3615347e-1dde-46b5-b2e3-af0300a049fa" | jq '.fields[] | select(.name == "Integration Key") | .value' --raw-output)
    ;;

  prod)
    # TODO: Fetch OCM token and log in as appropriate user as part of script.
    EXPECT_OCM_ID="2BBslbGSQs5PS2HCfJKqOPcCN4r"
    ACTUAL_OCM_ID=$(ocm whoami | jq -r '.id')
    if [[ "${EXPECT_OCM_ID}" != "${ACTUAL_OCM_ID}" ]]; then
      echo "Must be logged into rhacs-managed-service-prod account in OCM to get cluster ID"
      exit 1
    fi
    CLUSTER_ID=$(ocm list cluster "${CLUSTER_NAME}" --no-headers --columns="ID")

    FM_ENDPOINT="https://syqugpy2fa29zqc.api.openshift.com"

    ensure_bitwarden_session_exists
    # Note: the Red Hat SSO client as of 2022-09-02 is the same between stage and prod.
    FLEETSHARD_SYNC_RED_HAT_SSO_CLIENT_ID=$(bw get username 028ce1a9-f751-4056-9c72-aea70052728b)
    FLEETSHARD_SYNC_RED_HAT_SSO_CLIENT_SECRET=$(bw get password 028ce1a9-f751-4056-9c72-aea70052728b)
    LOGGING_AWS_ACCESS_KEY_ID=$(bw get item "f7711943-c355-47cc-a0ee-af0400f8dfe7" | jq '.fields[] | select(.name == "AccessKeyID") | .value' --raw-output)
    LOGGING_AWS_SECRET_ACCESS_KEY=$(bw get item "f7711943-c355-47cc-a0ee-af0400f8dfe7" | jq '.fields[] | select(.name == "SecretAccessKey") | .value' --raw-output)
    # Note: the GitHub Access Token as of 2022-09-02 is the same between stage and prod.
    OBSERVABILITY_GITHUB_ACCESS_TOKEN=$(bw get password eb7aecd3-b553-4999-b201-aebe01445822)
    OBSERVABILITY_OBSERVATORIUM_GATEWAY="https://observatorium-mst.api.openshift.com"
    OBSERVABILITY_OBSERVATORIUM_METRICS_CLIENT_ID="observatorium-rhacs-metrics"
    OBSERVABILITY_OBSERVATORIUM_METRICS_SECRET=$(
        bw get item 510c8ed9-ba9f-46d9-b906-ae6100cf72f5 | \
        jq --arg OBSERVABILITY_OBSERVATORIUM_METRICS_CLIENT_ID "${OBSERVABILITY_OBSERVATORIUM_METRICS_CLIENT_ID}" \
            '.fields[] | select(.name == $OBSERVABILITY_OBSERVATORIUM_METRICS_CLIENT_ID) | .value' --raw-output
    )
    # Note: the PagerDuty Service Key as of 2022-09-02 is the same between stage and prod.
    PAGERDUTY_SERVICE_KEY=$(bw get item "3615347e-1dde-46b5-b2e3-af0300a049fa" | jq '.fields[] | select(.name == "Integration Key") | .value' --raw-output)
    ;;

  *)
    echo "Unknown environment ${ENVIRONMENT}"
    exit 2
    ;;
esac

GIT_COMMIT_SHA=$(git rev-parse HEAD)
GIT_DESCRIBE=$(git describe --tag)

# helm template ... to debug changes
helm upgrade rhacs-terraform ./ \
  --install \
  --debug \
  --namespace rhacs \
  --create-namespace \
  --set acsOperator.enabled=true \
  --set acsOperator.source=redhat-operators \
  --set acsOperator.sourceNamespace=openshift-marketplace \
  --set acsOperator.startingCSV=rhacs-operator.v3.71.0 \
  --set fleetshardSync.authType="RHSSO" \
  --set fleetshardSync.gitCommitSHA="${GIT_COMMIT_SHA}" \
  --set fleetshardSync.gitDescribe="${GIT_DESCRIBE}" \
  --set fleetshardSync.clusterId=${CLUSTER_ID} \
  --set fleetshardSync.fleetManagerEndpoint=${FM_ENDPOINT} \
  --set fleetshardSync.redHatSSO.clientId="${FLEETSHARD_SYNC_RED_HAT_SSO_CLIENT_ID}" \
  --set fleetshardSync.redHatSSO.clientSecret="${FLEETSHARD_SYNC_RED_HAT_SSO_CLIENT_SECRET}" \
  --set logging.aws.accessKeyId="${LOGGING_AWS_ACCESS_KEY_ID}" \
  --set logging.aws.secretAccessKey="${LOGGING_AWS_SECRET_ACCESS_KEY}" \
  --set observability.github.accessToken="${OBSERVABILITY_GITHUB_ACCESS_TOKEN}" \
  --set observability.github.repository=https://api.github.com/repos/stackrox/rhacs-observability-resources/contents \
  --set observability.observatorium.gateway="${OBSERVABILITY_OBSERVATORIUM_GATEWAY}" \
  --set observability.observatorium.metricsClientId="${OBSERVABILITY_OBSERVATORIUM_METRICS_CLIENT_ID}" \
  --set observability.observatorium.metricsSecret="${OBSERVABILITY_OBSERVATORIUM_METRICS_SECRET}" \
  --set observability.pagerduty.key="${PAGERDUTY_SERVICE_KEY}"

# To uninstall an existing release:
# helm uninstall rhacs-terraform --namespace rhacs
#
# To delete all resources specified by the template:
# helm template ... > /var/tmp/resources.yaml
# kubectl delete -f /var/tmp/resources.yaml
