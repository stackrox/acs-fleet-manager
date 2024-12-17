#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# shellcheck source=scripts/lib/external_config.sh
source "$SCRIPT_DIR/../scripts/lib/external_config.sh"

if [[ $# -ne 2 ]]; then
    echo "Usage: $0 [environment] [cluster]" >&2
    echo "Known environments: integration stage prod"
    echo "Cluster typically looks like: acs-{env}-dp-01"
    echo "Description: This script will create identity providers for the OSD cluster:"
    echo "- OIDC provider using auth.redhat.com"
    echo "See additional documentation in docs/development/setup-osd-cluster-idp.md"
    echo
    echo "Note: you need to be logged into OCM for your environment's administrator"
    echo "Note: you need access to AWS account of the selected environment"
    exit 2
fi

ENVIRONMENT=$1
CLUSTER_NAME=$2

export AWS_AUTH_HELPER="${AWS_AUTH_HELPER:-aws-saml}"

export_cluster_environment() {
    init_chamber
    load_external_config "osd" OSD_
}

setup_oidc_provider() {
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
      echo "Skipping creating an OIDC IdP for the cluster, already exists."
    fi
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

export_cluster_environment
setup_oidc_provider

# This set of commands modifies OIDC provider to include "groups" claim mapping.
CLUSTER_IDP_ID=$(ocm get /api/clusters_mgmt/v1/clusters/"$CLUSTER_ID"/identity_providers | jq -r '.items[0].id')
tmpfile=$(mktemp /tmp/dataplane-idp-setup-tmp-patch-body.XXXXXX)
cat <<END >"$tmpfile"
{
  "type": "OpenIDIdentityProvider",
  "open_id": {
    "claims": {
      "email": [
        "email"
      ],
      "groups": [
        "groups"
      ],
      "name": [
        "preferred_username"
      ],
      "preferred_username": [
        "preferred_username"
      ]
    },
    "client_id": "${OSD_OIDC_CLIENT_ID}",
    "client_secret": "${OSD_OIDC_CLIENT_SECRET}",
    "issuer": "https://auth.redhat.com/auth/realms/EmployeeIDP"
  }
}
END
ocm patch /api/clusters_mgmt/v1/clusters/"$CLUSTER_ID"/identity_providers/"$CLUSTER_IDP_ID" --body="$tmpfile"
rm "$tmpfile"
