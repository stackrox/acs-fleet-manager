#!/usr/bin/env bash

## This script takes care of deploying Managed Service components.
GITROOT="$(git rev-parse --show-toplevel)"
export GITROOT
# shellcheck source=/dev/null
source "${GITROOT}/dev/env/scripts/lib.sh"
# shellcheck source=/dev/null
source "${GITROOT}/scripts/lib/external_config.sh"
# shellcheck source=/dev/null
source "${GITROOT}/dev/env/scripts/docker.sh"

init

if [[ "$ENABLE_EXTERNAL_CONFIG" == "true" ]]; then
    init_chamber
    export CHAMBER_SECRET_BACKEND=secretsmanager
else
    add_bin_to_path
    ensure_tool_installed chamber
    export CHAMBER_SECRET_BACKEND=null
fi


log "** Bringing up ACSCS **"
print_env

KUBE_CONFIG=$(assemble_kubeconfig | yq e . -o=json - | jq -c . -)
export KUBE_CONFIG

ensure_fleet_manager_image_exists
ensure_fleetshard_operator_image_exists

# Apply cluster type specific manifests, if any.
if [[ -d "${MANIFESTS_DIR}/cluster-type-${CLUSTER_TYPE}" ]]; then
    apply "${MANIFESTS_DIR}/cluster-type-${CLUSTER_TYPE}"
fi

# Deploy Cloud Service components.
log "Deploying secrets"
chamber exec "fleet-manager" -- make -C "$GITROOT" deploy/secrets

if ! is_openshift_cluster "$CLUSTER_TYPE"; then
    # These secrets are created in OpenShift by service-ca-operator
    # search for service.alpha.openshift.io/serving-cert-secret-name annotation.
    # We need at least empty secrets because they are referenced in the service template
    # but TLS is disabled for non-openshift clusters
    $KUBECTL -n "$ACSCS_NAMESPACE" create secret generic fleet-manager-tls 2> /dev/null || true
    $KUBECTL -n "$ACSCS_NAMESPACE" create secret generic fleet-manager-envoy-tls 2> /dev/null || true
    $KUBECTL -n "$ACSCS_NAMESPACE" create secret generic fleet-manager-active-tls 2> /dev/null || true
fi

DATAPLANE_ONLY=${DATAPLANE_ONLY:-}
if [[ -z "${DATAPLANE_ONLY}" ]]; then
    # Deploy database.
    log "Deploying database"
    make -C "$GITROOT" deploy/db
    wait_for_container_to_become_ready "$ACSCS_NAMESPACE" "application=fleet-manager-db" "postgresql"
    log "Database is ready."

    log "Deploying fleet-manager"
    make -C "$GITROOT" deploy/service
    wait_for_container_to_appear "$ACSCS_NAMESPACE" "app=fleet-manager" "service"

    if [[ "$SPAWN_LOGGER" == "true" && -n "${LOG_DIR:-}" ]]; then
    $KUBECTL -n "$ACSCS_NAMESPACE" logs -l app=fleet-manager --all-containers --pod-running-timeout=1m --since=1m --tail=100 -f >"${LOG_DIR}/pod-logs_fleet-manager.txt" 2>&1 &
    fi
fi

log "Deploying fleetshard-sync"
exec_fleetshard_sync.sh apply "${MANIFESTS_DIR}/fleetshard-sync"
apply "${MANIFESTS_DIR}/fleetshard-operator"

wait_for_container_to_appear "$ACSCS_NAMESPACE" "app=fleetshard-sync" "fleetshard-sync"
if [[ "$SPAWN_LOGGER" == "true" && -n "${LOG_DIR:-}" ]]; then
    $KUBECTL -n "$ACSCS_NAMESPACE" logs -l app=fleetshard-sync --all-containers --pod-running-timeout=1m --since=1m --tail=100 -f >"${LOG_DIR}/pod-logs_fleetshard-sync_fleetshard-sync.txt" 2>&1 &
fi

if [[ "$ENABLE_EMAIL_SENDER" == "true" ]]; then
    log "Deploying emailsender"
    make -C "$GITROOT" deploy/emailsender
    wait_for_container_to_appear "$ACSCS_NAMESPACE" "application=emailsender" "emailsender"
    if [[ "$SPAWN_LOGGER" == "true" && -n "${LOG_DIR:-}" ]]; then
        $KUBECTL -n "$ACSCS_NAMESPACE" logs -l application=emailsender --all-containers --pod-running-timeout=1m --since=1m --tail=100 -f >"${LOG_DIR}/pod-logs_emailsender_emailsender.txt" 2>&1 &
    fi
fi

# Sanity check.
wait_for_container_to_become_ready "$ACSCS_NAMESPACE" "app=fleetshard-sync" "fleetshard-sync" 500

if [[ -z "${DATAPLANE_ONLY}" ]]; then
    # Prerequisite for port-forwarding are pods in ready state.
    wait_for_container_to_become_ready "$ACSCS_NAMESPACE" "app=fleet-manager" "service"

    if [[ "$ENABLE_FM_PORT_FORWARDING" == "true" ]]; then
        log "Starting port-forwarding for fleet-manager"
        port-forwarding start fleet-manager 8000 8000
    else
        log "Skipping port-forwarding for fleet-manager"
    fi

    if [[ "$ENABLE_DB_PORT_FORWARDING" == "true" ]]; then
        log "Starting port-forwarding for db"
        port-forwarding start fleet-manager-db 5432 5432
    else
        log "Skipping port-forwarding for db"
    fi
fi

if [[ "$ENABLE_EMAIL_SENDER" == "true" ]]; then
    wait_for_container_to_become_ready "$ACSCS_NAMESPACE" "application=emailsender" "emailsender"
fi

log
log "** Fleet Manager ready ** "
log
