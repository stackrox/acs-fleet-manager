#!/usr/bin/env bash

export GITROOT="$(git rev-parse --show-toplevel)"
source "${GITROOT}/dev/env/scripts/lib.sh"
init

log "Tearing down deployment of MS components..."

port-forwarding stop fleet-manager || true
port-forwarding stop db || true

delete "${MANIFESTS_DIR}/rhacs-operator" || true
delete "${MANIFESTS_DIR}/db" || true
delete "${MANIFESTS_DIR}/fleet-manager" || true
delete "${MANIFESTS_DIR}/fleetshard-sync" || true
