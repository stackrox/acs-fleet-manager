#!/usr/bin/env bash

GITROOT="$(git rev-parse --show-toplevel)"
export GITROOT
# shellcheck source=/dev/null
source "${GITROOT}/dev/env/scripts/lib.sh"
init

log "Tearing down deployment of MS components..."

log "Stopping fleet-manager port-forwarding..."
port-forwarding stop fleet-manager 8000 || true

log "Stopping db port-forwarding..."
port-forwarding stop fleet-manager-db 5432 || true

log "Cleanup files..."
make -C "${GITROOT}" undeploy undeploy/fleetshard-sync

log "Cleanup namespaces..."
delete_tenant_namespaces

$KUBECTL -n "${STACKROX_OPERATOR_NAMESPACE}" delete deploy -l app=rhacs-operator || true
