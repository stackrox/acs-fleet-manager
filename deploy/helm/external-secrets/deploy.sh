#!/usr/bin/env bash
set -euo pipefail

VERSION=0.8.3
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# shellcheck source=scripts/lib/external_config.sh
source "$SCRIPT_DIR/../../../scripts/lib/external_config.sh"
# shellcheck source=scripts/lib/helm.sh
source "$SCRIPT_DIR/../../../scripts/lib/helm.sh"

if [[ $# -ne 2 ]]; then
    echo "Usage: $0 [environment] [cluster]" >&2
    echo "Known environments: dev stage prod"
    echo "Cluster typically looks like: acs-{environment}-dp-01"
    exit 2
fi

ENVIRONMENT=$1
CLUSTER_NAME=$2

if [[ "${ENVIRONMENT}" != "dev" ]]; then
    init_chamber
    load_external_config "cluster-${CLUSTER_NAME}" CLUSTER_
    oc login --token="${CLUSTER_ROBOT_OC_TOKEN}" --server="$CLUSTER_URL"
fi

helm repo add external-secrets https://charts.external-secrets.io

helm upgrade -i external-secrets \
   external-secrets/external-secrets \
    --version "${VERSION}" \
    -n external-secrets \
    --create-namespace \
    --values "$SCRIPT_DIR/values.yaml" \
    --set "serviceAccount.annotations.eks\.amazonaws\.com/role-arn=arn:aws:iam::${AWS_ACCOUNT_ID}:role/ExternalSecretsServiceRole"
