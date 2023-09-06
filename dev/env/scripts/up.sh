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

if [[ "$IGNORE_REPOSITORY_DIRTINESS" = "true" ]]; then
    fleet_manager_image_info="${FLEET_MANAGER_IMAGE} (ignoring repository dirtiness)"
else
    fleet_manager_image_info="${FLEET_MANAGER_IMAGE}"
fi

cat <<EOF

** Bringing up ACS MS **

Image: ${fleet_manager_image_info}
Cluster Name: ${CLUSTER_NAME}
Cluster Type: ${CLUSTER_TYPE}
Namespace: ${ACSMS_NAMESPACE}

Inheriting ImagePullSecrets for Quay.io: ${INHERIT_IMAGEPULLSECRETS}
Installing RHACS Operator: ${INSTALL_OPERATOR}
Enable External Config: ${ENABLE_EXTERNAL_CONFIG}
AWS Auth Helper: ${AWS_AUTH_HELPER:-none}
Debugging Mode: ${DEBUG_PODS}

EOF

KUBE_CONFIG=$(assemble_kubeconfig | yq e . -o=json - | jq -c . -)
export KUBE_CONFIG

ensure_fleet_manager_image_exists

# Apply cluster type specific manifests, if any.
if [[ -d "${MANIFESTS_DIR}/cluster-type-${CLUSTER_TYPE}" ]]; then
    apply "${MANIFESTS_DIR}/cluster-type-${CLUSTER_TYPE}"
fi

# Deploy database.
log "Deploying database"
apply "${MANIFESTS_DIR}/db"
wait_for_container_to_become_ready "$ACSMS_NAMESPACE" "application=db" "db"
log "Database is ready."

# Deploy MS components.
log "Deploying fleet-manager"
chamber exec "fleet-manager" -- apply "${MANIFESTS_DIR}/fleet-manager"
inject_exported_env_vars "$ACSMS_NAMESPACE" "fleet-manager"

wait_for_container_to_appear "$ACSMS_NAMESPACE" "application=fleet-manager" "fleet-manager"
if [[ "$SPAWN_LOGGER" == "true" && -n "${LOG_DIR:-}" ]]; then
    $KUBECTL -n "$ACSMS_NAMESPACE" logs -l application=fleet-manager --all-containers --pod-running-timeout=1m --since=1m --tail=100 -f >"${LOG_DIR}/pod-logs_fleet-manager.txt" 2>&1 &
fi

log "Deploying fleetshard-sync"
exec_fleetshard_sync.sh apply "${MANIFESTS_DIR}/fleetshard-sync"
inject_exported_env_vars "$ACSMS_NAMESPACE" "fleetshard-sync"

wait_for_container_to_appear "$ACSMS_NAMESPACE" "application=fleetshard-sync" "fleetshard-sync"
if [[ "$SPAWN_LOGGER" == "true" && -n "${LOG_DIR:-}" ]]; then
    $KUBECTL -n "$ACSMS_NAMESPACE" logs -l application=fleetshard-sync --all-containers --pod-running-timeout=1m --since=1m --tail=100 -f >"${LOG_DIR}/pod-logs_fleetshard-sync_fleetshard-sync.txt" 2>&1 &
fi

# Sanity check.
wait_for_container_to_become_ready "$ACSMS_NAMESPACE" "application=fleetshard-sync" "fleetshard-sync" 500
# Prerequisite for port-forwarding are pods in ready state.
wait_for_container_to_become_ready "$ACSMS_NAMESPACE" "application=fleet-manager" "fleet-manager"

if [[ "$ENABLE_FM_PORT_FORWARDING" == "true" ]]; then
    port-forwarding start fleet-manager 8000 8000
fi

if [[ "$ENABLE_DB_PORT_FORWARDING" == "true" ]]; then
    port-forwarding start db 5432 5432
fi

log
log "** Fleet Manager ready ** "
log
