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
    echo "It will NOT create a ServiceAccount for the data plane continuous deployment."
    echo "See the cd-robot-account-setup.sh for that."
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

    # Create the users that should have access to the cluster with cluster administrative rights.
    # Ignore errors as the sometimes users already exist.
    ocm create user --cluster="${CLUSTER_NAME}" \
      --group=cluster-admins \
      "${OSD_OIDC_USER_LIST}" || true
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

# This command grants access to all RH employees to access cluster monitoring.
oc apply -f - <<END
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: acs-general-observability
subjects:
  - kind: Group
    apiGroup: rbac.authorization.k8s.io
    name: Employee
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-monitoring-view
END
