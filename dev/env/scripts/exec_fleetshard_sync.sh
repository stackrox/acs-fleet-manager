#!/usr/bin/env bash

set -euo pipefail

GITROOT="$(git rev-parse --show-toplevel)"
ENABLE_EXTERNAL_CONFIG="${ENABLE_EXTERNAL_CONFIG:-true}"

ARGS="${GITROOT}/fleetshard-sync"
if [[ "$#" -gt 0 ]]; then
    ARGS="$*"
fi

if [[ "$ENABLE_EXTERNAL_CONFIG" != "true" ]]; then
    ${ARGS}
    exit
fi

export AWS_AUTH_HELPER="${AWS_AUTH_HELPER:-aws-saml}"
# shellcheck source=scripts/lib/external_config.sh
source "${GITROOT}/scripts/lib/external_config.sh"
init_chamber

CLUSTER_NAME="${CLUSTER_NAME:-cluster-acs-dev-dp-01}"

ARGS="CLUSTER_ID=${CLUSTER_ID:-$(chamber read "${CLUSTER_NAME}" ID -q -b ssm)} \
    MANAGED_DB_SECURITY_GROUP=${MANAGED_DB_SECURITY_GROUP:-$(chamber read "${CLUSTER_NAME}" MANAGED_DB_SECURITY_GROUP -q -b ssm)} \
    MANAGED_DB_SUBNET_GROUP=${MANAGED_DB_SUBNET_GROUP:-$(chamber read "${CLUSTER_NAME}" MANAGED_DB_SUBNET_GROUP -q -b ssm)} \
    SECRET_ENCRYPTION_KEY_ID=${SECRET_ENCRYPTION_KEY_ID:-$(chamber read "${CLUSTER_NAME}" SECRET_ENCRYPTION_KEY_ID -q -b ssm)} \
    AWS_ROLE_ARN=${FLEETSHARD_SYNC_AWS_ROLE_ARN:-$(chamber read fleetshard-sync AWS_ROLE_ARN -q -b ssm)} \
    $ARGS"

chamber exec fleetshard-sync -b secretsmanager -- sh -c "$ARGS"
