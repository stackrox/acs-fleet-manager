#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# shellcheck source=scripts/lib/external_config.sh
source "$SCRIPT_DIR/../scripts/lib/external_config.sh"

if [[ $# -ne 2 ]]; then
    echo "Usage: $0 [environment] [cluster]" >&2
    echo "Known environments: stage prod"
    echo "Cluster typically looks like: acs-{environment}-dp-01"
    echo "Description: This script will create an identity provider for the OSD cluster, based on the environment it will be OIDC provider using auth.redhat.com (for stage) or HTPasswd provider (for prod)"
    echo "Note: you need to be logged into OCM for your environment's administrator"
    echo "Note: you need access to BitWarden"
    exit 2
fi

ENVIRONMENT=$1
CLUSTER_NAME=$2

export AWS_PROFILE="$ENVIRONMENT"

case $ENVIRONMENT in
  stage)
    EXPECT_OCM_ID="2ECw6PIE06TzjScQXe6QxMMt3Sa"
    ACTUAL_OCM_ID=$(ocm whoami | jq -r '.id')
    if [[ "${EXPECT_OCM_ID}" != "${ACTUAL_OCM_ID}" ]]; then
      echo "Must be logged into rhacs-managed-service-stage account in OCM to get cluster ID"
      exit 1
    fi
    CLUSTER_ID=$(ocm list cluster "${CLUSTER_NAME}" --no-headers --columns="ID")

    # Load configuration
    init_chamber
    load_external_config "cluster-$CLUSTER_NAME"

    # Create the IdP for the cluster.
    ocm create idp --name=OpenID \
      --cluster="${CLUSTER_ID}" \
      --type=openid \
      --client-id="${OSD_OIDC_CLIENT_ID}" \
      --client-secret="${OSD_OIDC_CLIENT_SECRET}" \
      --issuer-url=https://auth.redhat.com/auth/realms/EmployeeIDP \
      --email-claims=email --name-claims=preferred_username --username-claims=preferred_username

    # Create the users that should have access to the cluster with cluster administrative rights.
    # Ignore errors as the sometimes users already exist.
    ocm create user --cluster="${CLUSTER_NAME}" \
      --group=cluster-admins \
      "${OSD_OIDC_USER_LIST}" || true

    ocm create idp --name=HTPasswd \
      --cluster="${CLUSTER_ID}" \
      --type=htpasswd \
      --username="${ADMIN_USERNAME}" \
      --password="${ADMIN_PASSWORD}"

    # Create the acsms-admin user. Ignore errors, if it already exists.
    ocm create user --cluster="${CLUSTER_NAME}" \
      --group=cluster-admins \
      "${ADMIN_USERNAME}" || true

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

    # Load configuration
    init_chamber
    load_external_config "cluster-$CLUSTER_NAME"

    # Create the IdP for the cluster.
    ocm create idp --name=HTPasswd \
      --cluster="${CLUSTER_ID}" \
      --type=htpasswd \
      --username="${ADMIN_USERNAME}" \
      --password="${ADMIN_PASSWORD}"

    # Create the acsms-admin user. Ignore errors, if it already exists.
    ocm create user --cluster="${CLUSTER_NAME}" \
      --group=cluster-admins \
      "${ADMIN_USERNAME}" || true
    ;;

  *)
    echo "Unknown environment ${ENVIRONMENT}"
    exit 2
    ;;
esac
