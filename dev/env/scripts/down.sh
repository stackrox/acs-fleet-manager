#!/usr/bin/env bash

GITROOT="$(git rev-parse --show-toplevel)"
export GITROOT
# shellcheck source=/dev/null
source "${GITROOT}/dev/env/scripts/lib.sh"
init

log "Tearing down deployment of MS components..."

port-forwarding stop fleet-manager 8000 || true
port-forwarding stop db 5432 || true

delete "${MANIFESTS_DIR}/db" || true
delete "${MANIFESTS_DIR}/fleet-manager" || true
delete "${MANIFESTS_DIR}/fleetshard-sync" || true

central_namespaces=$($KUBECTL get namespace -o jsonpath='{range .items[?(@.status.phase == "Active")]}{.metadata.name}{"\n"}{end}' | grep '^rhacs-.*$')

for namespace in $central_namespaces; do
    $KUBECTL delete namespace "$namespace" &
done
log "Waiting for leftover RHACS namespaces to be deleted... "
for p in $(jobs -pr); do
    wait "$p"
done
