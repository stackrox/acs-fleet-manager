#!/usr/bin/env bash

set -euo pipefail

GITROOT="$(git rev-parse --show-toplevel)"

ARGS="${GITROOT}/fleetshard-sync"
if [[ "$#" -gt 0 ]]; then
    ARGS="$*"
fi

CLUSTER_NAME="${CLUSTER_NAME:-cluster-acs-dev-dp-01}"

ARGS="CLUSTER_ID=local_cluster \
    $ARGS"
