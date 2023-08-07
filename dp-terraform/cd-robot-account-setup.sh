#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# shellcheck source=scripts/lib/external_config.sh
source "$SCRIPT_DIR/../scripts/lib/external_config.sh"

if [[ $# -ne 2 ]]; then
    echo "Usage: $0 [environment] [cluster]" >&2
    echo "Known environments: integration stage prod"
    echo "Cluster typically looks like: acs-{env}-dp-01"
    echo "Description:"
    echo "This script will create and configure a ServiceAccount for the data plane continuous deployment."
    echo "It should be safe to run this script many times, it should be idempotent."
    echo
    echo "Note: you need to be logged into OCM for your environment's administrator"
    echo "Note: you need access to AWS account of the selected environment"
    exit 2
fi

ENVIRONMENT=$1
CLUSTER_NAME=$2

export AWS_AUTH_HELPER="${AWS_AUTH_HELPER:-aws-saml}"

save_cluster_parameter() {
    local key="$1"
    local value="$2"
    echo "Saving parameter '/cluster-${CLUSTER_NAME}/${key}' in AWS parameter store..."
    chamber write "cluster-${CLUSTER_NAME}" "${key}" "${value}" --skip-unchanged
}

save_cluster_secret() {
    local key="$1"
    local value="$2"
    echo "Saving parameter '/cluster-${CLUSTER_NAME}/${key}' in AWS Secrets Manager..."
    chamber write -b secretsmanager "cluster-${CLUSTER_NAME}" "${key}" "${value}" --skip-unchanged
}

case $ENVIRONMENT in
  integration)
    EXPECT_OCM_ID="2QVFzUvsbMGheHhoUDjtG0tpJ08"
    ;;

  stage)
    EXPECT_OCM_ID="2ECw6PIE06TzjScQXe6QxMMt3Sa"
    ;;

  prod)
    # TODO: Fetch OCM token and log in as appropriate user as part of script.
    EXPECT_OCM_ID="2BBslbGSQs5PS2HCfJKqOPcCN4r"
    ;;

  *)
    echo "Unknown environment ${ENVIRONMENT}"
    exit 2
    ;;
esac

ACTUAL_OCM_ID=$(ocm whoami | jq -r '.id')
if [[ "${EXPECT_OCM_ID}" != "${ACTUAL_OCM_ID}" ]]; then
  echo "Must be logged into rhacs-managed-service-$ENVIRONMENT account in OCM to get cluster ID"
  exit 1
fi
CLUSTER_ID=$(ocm list cluster "${CLUSTER_NAME}" --no-headers --columns="ID")

init_chamber

# Retrieve the cluster token from the configured IdP interactively.
echo "Login to the cluster using the OIDC IdP and obtain a token."
ocm cluster login "${CLUSTER_NAME}" --token
# This requires users to paste the token, since the command only opens the browser but doesn't retrieve the token itself.
echo "Paste the token (it will not be echoed to the screen):"
read -r -s CLUSTER_TOKEN

# The ocm command likes to return trailing whitespace, so try and trim it:
CLUSTER_URL="$(ocm list cluster "${CLUSTER_NAME}" --no-headers --columns api.url | awk '{print $1}')"

# Use a temporary KUBECONFIG to avoid storing credentials in and changing current context in user's day-to-day kubeconfig.
KUBECONFIG="$(mktemp)"
export KUBECONFIG
trap 'rm -f "${KUBECONFIG}"' EXIT

echo "Logging into cluster ${CLUSTER_NAME}..."
oc login "${CLUSTER_URL}" --token="${CLUSTER_TOKEN}"

ROBOT_NS="acscs-dataplane-cd"
ROBOT_SA="acscs-cd-robot"
ROBOT_TOKEN_RESOURCE="robot-token"

echo "Provisioning robot account and configuring its permissions..."
# We use `apply` rather than `create` for idempotence.
oc apply -f - <<END
apiVersion: v1
kind: Namespace
metadata:
  name: ${ROBOT_NS}
END
oc apply -f - <<END
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ${ROBOT_SA}
  namespace: ${ROBOT_NS}
END
oc adm policy -n "${ROBOT_NS}" --rolebinding-name="acscs-cd-robot-admin" add-cluster-role-to-user cluster-admin -z "${ROBOT_SA}"
oc apply -n "${ROBOT_NS}" -f - <<END
apiVersion: v1
kind: Secret
metadata:
  name: ${ROBOT_TOKEN_RESOURCE}
  annotations:
    kubernetes.io/service-account.name: "${ROBOT_SA}"
type: kubernetes.io/service-account-token
END

save_cluster_parameter "id" "$CLUSTER_ID"
save_cluster_parameter "url" "$CLUSTER_URL"

echo "Polling for token to be provisioned."
attempt=0
while true
do
  attempt=$((attempt+1))
  ROBOT_TOKEN="$(oc get secret "${ROBOT_TOKEN_RESOURCE}" -n "$ROBOT_NS" -o json | jq -r 'if (has("data") and (.data|has("token"))) then (.data.token|@base64d) else "" end')"
  if [[ -n $ROBOT_TOKEN ]]; then
    save_cluster_secret "robot_oc_token" "$ROBOT_TOKEN"
    break
  fi
  if [[ $attempt -gt 30 ]]; then
    echo "Timed out waiting for a token to be provisioned in the ${ROBOT_TOKEN_RESOURCE} secret."
    exit 1
  fi
  sleep 1
done

echo "The following cluster parameters are currently stored in AWS Parameter Store:"
chamber list "cluster-${CLUSTER_NAME}"
echo "The following cluster parameters are currently stored in AWS Secrets Manager:"
chamber list "cluster-${CLUSTER_NAME}" -b secretsmanager
