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
    return
fi

export AWS_AUTH_HELPER="${AWS_AUTH_HELPER:-aws-vault}"
# shellcheck source=scripts/lib/external_config.sh
source "${GITROOT}/scripts/lib/external_config.sh"
init_chamber

CLUSTER_NAME="cluster-acs-dev-dp-01"

ARGS="CLUSTER_ID=$(run_chamber read ${CLUSTER_NAME} ID -q -b ssm) \
    MANAGED_DB_SECURITY_GROUP=$(run_chamber read ${CLUSTER_NAME} MANAGED_DB_SECURITY_GROUP -q -b ssm) \
    MANAGED_DB_SUBNET_GROUP=$(run_chamber read ${CLUSTER_NAME} MANAGED_DB_SUBNET_GROUP -q -b ssm) \
    AWS_ROLE_ARN=$(run_chamber read fleetshard-sync AWS_ROLE_ARN -q -b ssm) \
    $ARGS"

run_chamber exec fleetshard-sync -b secretsmanager -- sh -c "$ARGS"
