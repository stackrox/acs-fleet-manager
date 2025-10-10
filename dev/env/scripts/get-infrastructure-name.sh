#!/usr/bin/env bash

# This script retrieves the Infrastructure CR's infrastructureName from the cluster
# and exports it as INFRASTRUCTURE_NAME for use in manifest templating.

set -euo pipefail

GITROOT="$(git rev-parse --show-toplevel)"
export GITROOT
# shellcheck source=/dev/null
source "${GITROOT}/scripts/lib/log.sh"

KUBECTL_BIN=${KUBECTL:-kubectl}

INFRASTRUCTURE_NAME=$($KUBECTL_BIN get infrastructures.config.openshift.io cluster -o jsonpath='{.status.infrastructureName}')

if [[ -z "$INFRASTRUCTURE_NAME" ]]; then
    die "Error: Could not retrieve infrastructure name from cluster"
fi

export INFRASTRUCTURE_NAME
log "Infrastructure name: $INFRASTRUCTURE_NAME"
