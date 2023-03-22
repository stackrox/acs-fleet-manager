#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# shellcheck source=scripts/lib/external_config.sh
source "$SCRIPT_DIR/../scripts/lib/external_config.sh"

if [[ $# -ne 2 ]]; then
    echo "Usage: $0 [environment] [cluster]" >&2
    echo "Known environments: stage prod"
    echo "Cluster typically looks like: acs-{environment}-dp-01"
    echo "Description: This script will create an identity provider for the OSD cluster, based on the environment it will be:"
    echo "- stage: OIDC provider using auth.redhat.com and HTPasswd provider"
    echo "- prod: HTPasswd provider"
    echo "It will also create and configure a ServiceAccount for the data plane continuous deployment."
    echo "See additional documentation in docs/development/setup-osd-cluster-idp.md"
    echo
    echo "Note: you need to be logged into OCM for your environment's administrator"
    echo "Note: you need access to AWS account of the selected environment"
    exit 2
fi

ENVIRONMENT=$1
CLUSTER_NAME=$2

export AWS_AUTH_HELPER="${AWS_AUTH_HELPER:-aws-saml}"
if [[ "$AWS_AUTH_HELPER" == "aws-vault" ]]; then
    export AWS_PROFILE="$ENVIRONMENT"
fi

save_cluster_parameter() {
    local key="$1"
    local value="$2"
    echo "Saving parameter '/cluster-${CLUSTER_NAME}/${key}' in AWS parameter store..."
    run_chamber write "cluster-${CLUSTER_NAME}" "${key}" "${value}" --skip-unchanged
}

save_cluster_secret() {
    local key="$1"
    local value="$2"
    echo "Saving parameter '/cluster-${CLUSTER_NAME}/${key}' in AWS Secrets Manager..."
    run_chamber write -b secretsmanager "cluster-${CLUSTER_NAME}" "${key}" "${value}" --skip-unchanged
}

export_cluster_environment() {
    init_chamber
    load_external_config "osd" OSD_
    load_external_config "cluster-$CLUSTER_NAME" STORED_

    if [[ -z ${STORED_ADMIN_USERNAME:-} ]]; then
        echo "Cluster admin user not specified in Secrets Manager nor Parameter Store."
        echo "Enter cluster admin username:"
        read -r STORED_ADMIN_USERNAME
        save_cluster_secret "admin_username" "$STORED_ADMIN_USERNAME"
    fi
    if [[ -z ${STORED_ADMIN_PASSWORD:-} ]]; then
        echo "Enter cluster admin password:"
        read -r -s STORED_ADMIN_PASSWORD
        save_cluster_secret "admin_password" "$STORED_ADMIN_PASSWORD"
    fi
}

case $ENVIRONMENT in
  stage)
    EXPECT_OCM_ID="2ECw6PIE06TzjScQXe6QxMMt3Sa"
    ACTUAL_OCM_ID=$(ocm whoami | jq -r '.id')
    if [[ "${EXPECT_OCM_ID}" != "${ACTUAL_OCM_ID}" ]]; then
      echo "Must be logged into rhacs-managed-service-stage account in OCM to get cluster ID"
      exit 1
    fi
    CLUSTER_ID=$(ocm list cluster "${CLUSTER_NAME}" --no-headers --columns="ID")

    export_cluster_environment

    if ! ocm list idps --cluster="${CLUSTER_NAME}" --columns name | grep -qE '^OpenID *$'; then
      echo "Creating an OpenID IdP for the cluster."
      ocm create idp --name=OpenID \
        --cluster="${CLUSTER_ID}" \
        --type=openid \
        --client-id="${OSD_OIDC_CLIENT_ID}" \
        --client-secret="${OSD_OIDC_CLIENT_SECRET}" \
        --issuer-url=https://auth.redhat.com/auth/realms/EmployeeIDP \
        --email-claims=email --name-claims=preferred_username --username-claims=preferred_username
    else
      echo "Skipping creation an OpenID IdP for the cluster, already exists."
    fi

    # Create the users that should have access to the cluster with cluster administrative rights.
    # Ignore errors as the sometimes users already exist.
    ocm create user --cluster="${CLUSTER_NAME}" \
      --group=cluster-admins \
      "${OSD_OIDC_USER_LIST}" || true

    if ! ocm list idps --cluster="${CLUSTER_NAME}" --columns name | grep -qE '^HTPasswd *$'; then
      echo "Creating an HTPasswd IdP for the cluster."
      ocm create idp --name=HTPasswd \
        --cluster="${CLUSTER_ID}" \
        --type=htpasswd \
        --username="${STORED_ADMIN_USERNAME}" \
        --password="${STORED_ADMIN_PASSWORD}"
    else
      echo "Skipping creation an HTPasswd IdP for the cluster, already exists."
    fi

    # Create the acsms-admin user. Ignore errors, if it already exists.
    ocm create user --cluster="${CLUSTER_NAME}" \
      --group=cluster-admins \
      "${STORED_ADMIN_USERNAME}" || true

    ;;

  prod)
    # For production environment, the OIDC client we currently have is not yet suitable (we have to order one per environment)
    # TODO(dhaus): once we have the  production client, add those values here.
    echo "For production, the OIDC client is not yet available. Still using the HTPasswd client for this"

    # TODO: Fetch OCM token and log in as appropriate user as part of script.
    EXPECT_OCM_ID="2BBslbGSQs5PS2HCfJKqOPcCN4r"
    ACTUAL_OCM_ID=$(ocm whoami | jq -r '.id')
    if [[ "${EXPECT_OCM_ID}" != "${ACTUAL_OCM_ID}" ]]; then
      echo "Must be logged into rhacs-managed-service-prod account in OCM to get cluster ID"
      exit 1
    fi
    CLUSTER_ID=$(ocm list cluster "${CLUSTER_NAME}" --no-headers --columns="ID")

    export_cluster_environment

    if ! ocm list idps --cluster="${CLUSTER_NAME}" --columns name | grep -qE '^HTPasswd *$'; then
      echo "Creating an HTPasswd IdP for the cluster."
      ocm create idp --name=HTPasswd \
        --cluster="${CLUSTER_ID}" \
        --type=htpasswd \
        --username="${STORED_ADMIN_USERNAME}" \
        --password="${STORED_ADMIN_PASSWORD}"
    else
      echo "Skipping creation an HTPasswd IdP for the cluster, already exists."
    fi

    # Create the acsms-admin user. Ignore errors, if it already exists.
    ocm create user --cluster="${CLUSTER_NAME}" \
      --group=cluster-admins \
      "${STORED_ADMIN_USERNAME}" || true
    ;;

  *)
    echo "Unknown environment ${ENVIRONMENT}"
    exit 2
    ;;
esac

# The ocm command likes to return trailing whitespace, so try and trim it:
CLUSTER_URL="$(ocm list cluster "${CLUSTER_NAME}" --no-headers --columns api.url | awk '{print $1}')"

# Use a temporary KUBECONFIG to avoid storing credentials in and changing current context in user's day-to-day kubeconfig.
KUBECONFIG="$(mktemp)"
export KUBECONFIG
trap 'rm -f "${KUBECONFIG}"' EXIT

echo "Logging into cluster ${CLUSTER_NAME} as ${STORED_ADMIN_USERNAME}..."
oc login "${CLUSTER_URL}" --username="${STORED_ADMIN_USERNAME}" --password="${STORED_ADMIN_PASSWORD}"

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
run_chamber list "cluster-${CLUSTER_NAME}"
echo "The following cluster parameters are currently stored in AWS Secrets Manager:"
run_chamber list "cluster-${CLUSTER_NAME}" -b secretsmanager
