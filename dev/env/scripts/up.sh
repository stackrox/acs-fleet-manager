#!/usr/bin/env bash

## This script takes care of deploying Managed Service components.

GITROOT="$(git rev-parse --show-toplevel)"
export GITROOT
# shellcheck source=/dev/null
source "${GITROOT}/dev/env/scripts/lib.sh"
init

cat <<EOF

** Bringing up ACS MS **

Image: ${FLEET_MANAGER_IMAGE}
Namespace: ${ACSMS_NAMESPACE}
Inheriting ImagePullSecrets for Quay.io: ${INHERIT_IMAGEPULLSECRETS}
Installing RHACS Operator: ${INSTALL_OPERATOR}

EOF

KUBE_CONFIG=$(assemble_kubeconfig | yq e . -j - | jq -c . -)
export KUBE_CONFIG

if [[ "$CLUSTER_TYPE" != "openshift-ci" ]]; then
    if [[ "$FLEET_MANAGER_IMAGE" =~ ^fleet-manager:.* ]]; then
        # Local image reference, which cannot be pulled.
        image_available=$(if $DOCKER image inspect "${FLEET_MANAGER_IMAGE}" >/dev/null 2>&1; then echo "true"; else echo "false"; fi)
        if [[ "$image_available" != "true" || "$FLEET_MANAGER_IMAGE" =~ dirty$ ]]; then
            # Attempt to build this image.
            if [[ "$FLEET_MANAGER_IMAGE" == "fleet-manager:$(make -s -C "${GITROOT}" tag)" ]]; then
                # Looks like we can build this tag from the current state of the repository.
                log "Rebuilding image..."
                make -C "${GITROOT}" image/build/local
            else
                die "Cannot find image '${FLEET_MANAGER_IMAGE}' and don't know how to build it"
            fi
        fi
    else
        log "Trying to pull image '${FLEET_MANAGER_IMAGE}'..."
        $DOCKER pull "$FLEET_MANAGER_IMAGE"
    fi
fi

# Verify that the image is there.
if ! $DOCKER image inspect "$FLEET_MANAGER_IMAGE" >/dev/null 2>&1; then
    die "Image ${FLEET_MANAGER_IMAGE} not available in cluster, aborting"
fi

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
apply "${MANIFESTS_DIR}/fleet-manager"
wait_for_container_to_appear "$ACSMS_NAMESPACE" "application=fleet-manager" "db-migrate"
wait_for_container_to_appear "$ACSMS_NAMESPACE" "application=fleet-manager" "fleet-manager"
if [[ "$SPAWN_LOGGER" == "true" && -n "${LOG_DIR:-}" ]]; then
    $KUBECTL -n "$ACSMS_NAMESPACE" logs -l application=fleet-manager --all-containers --pod-running-timeout=1m --since=1m --tail=100 -f >"${LOG_DIR}/pod-logs_fleet-manager.txt" 2>&1 &
fi

log "Deploying fleetshard-sync"
apply "${MANIFESTS_DIR}/fleetshard-sync"
wait_for_container_to_appear "$ACSMS_NAMESPACE" "application=fleetshard-sync" "fleetshard-sync"
if [[ "$SPAWN_LOGGER" == "true" && -n "${LOG_DIR:-}" ]]; then
    $KUBECTL -n "$ACSMS_NAMESPACE" logs -l application=fleetshard-sync --all-containers --pod-running-timeout=1m --since=1m --tail=100 -f >"${LOG_DIR}/pod-logs_fleetshard-sync_fleetshard-sync.txt" 2>&1 &
fi

# Sanity check.
wait_for_container_to_become_ready "$ACSMS_NAMESPACE" "application=fleetshard-sync" "fleetshard-sync"
# Prerequisite for port-forwarding are pods in ready state.
wait_for_container_to_become_ready "$ACSMS_NAMESPACE" "application=fleet-manager" "fleet-manager"

if [[ "$ENABLE_FM_PORT_FORWARDING" == "true" ]]; then
    port-forwarding start fleet-manager 8000 8000
fi

if [[ "$ENABLE_DB_PORT_FORWARDING" == "true" ]]; then
    port-forwarding start db 5432 5432
fi

log "** Fleet Manager ready ** "
