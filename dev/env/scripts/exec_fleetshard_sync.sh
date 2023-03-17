#!/usr/bin/env bash

set -euo pipefail

CLUSTER_KEY="dev/acscs/cluster-acs-dev-dp-01"
GITROOT="$(git rev-parse --show-toplevel)"

export AWS_AUTH_HELPER="${AWS_AUTH_HELPER:-aws-vault}"

# shellcheck source=scripts/lib/external_config.sh
source "${GITROOT}/scripts/lib/external_config.sh"
init_chamber

ARGS="${GITROOT}/fleetshard-sync"
if [[ "$#" -gt 0 ]]; then
    ARGS="$*"
fi
# shellcheck disable=SC2086
CLUSTER_ID=$(run_chamber read "${CLUSTER_KEY}" ID -q) \
MANAGED_DB_SECURITY_GROUP=$(run_chamber read "${CLUSTER_KEY}" MANAGED_DB_SECURITY_GROUP -q) \
MANAGED_DB_SUBNET_GROUP=$(run_chamber read "${CLUSTER_KEY}" MANAGED_DB_SUBNET_GROUP -q) \
run_chamber exec dev/acscs/fleetshard-sync -b secretsmanager -- ${ARGS}
