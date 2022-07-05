#!/usr/bin/env bash

## This script takes care of deploying Managed Service components.

export GITROOT="$(git rev-parse --show-toplevel)"
source "${GITROOT}/dev/env/scripts/lib.sh"
init

cat <<EOF
** Bringing up ACS MS **

Image: ${FLEET_MANAGER_IMAGE}
Namespace: ${ACSMS_NAMESPACE}
Inheriting ImagePullSecrets for Quay.io: ${INHERIT_IMAGEPULLSECRETS}
Installing RHACS Operator: ${INSTALL_OPERATOR}

EOF

if [[ -z "$OPENSHIFT_CI" ]]; then
    if [[ "$FLEET_MANAGER_IMAGE" =~ ^fleet-manager-.*/fleet-manager:.* ]]; then
        # Local image reference, which cannot be pulled.
        if ! $DOCKER image inspect "${FLEET_MANAGER_IMAGE}" >/dev/null 2>&1; then
            # Attempt to build this image.
            if [[ "$FLEET_MANAGER_IMAGE" == "$(make -C "${GITROOT}" image-tag)" ]]; then
                # Looks like we can build this tag from the current state of the repository.
                log "Rebuilding image..."
                make -C "${GITROOT}" image/build
            else
                die "Cannot find image '${FLEET_MANAGER_IMAGE}' and don't know how to build it"
            fi
        fi
    else
        log "Trying to pull image '${FLEET_MANAGER_IMAGE}'..."
        $DOCKER pull "$FLEET_MANAGER_IMAGE"
    fi
fi

if [[ "$CLUSTER_TYPE" == "minikube" ]]; then
    # Workaround for a bug in minikube(?) where sometimes the images fail to load:
    log "Deleting docker containers running in Minikube"
    $MINIKUBE ssh 'docker kill $(docker ps -q) > /dev/null' || true
    sleep 1
    $MINIKUBE ssh 'docker rm --force $(docker ps -a -q) > /dev/null' || true
    sleep 1
    $MINIKUBE image ls | grep -v "^${FLEET_MANAGER_IMAGE}$" | { grep "^.*/fleet-manager-.*/fleet-manager:.*$" || test $? = 1; } | while read img; do
        $MINIKUBE image rm "$img" || true
    done
    # In a perfect world this line would be sufficient...
    $DOCKER save "${FLEET_MANAGER_IMAGE}" | $MINIKUBE ssh --native-ssh=false docker load
    $MINIKUBE image ls | grep -q "^${FULL_FLEET_MANAGER_IMAGE}$" || {
        # Double check the image is there -- has been failing often enough due to the bug with the above workaround.
        die "Image ${FULL_FLEET_MANAGER_IMAGE} not available in cluster, aborting"
    }
fi

# Apply cluster type specific manifests.
if [[ -d "${MANIFESTS_DIR}/cluster-type-${CLUSTER_TYPE}" ]]; then
    apply "${MANIFESTS_DIR}/cluster-type-${CLUSTER_TYPE}"
fi

# Deploy database.
apply "${MANIFESTS_DIR}/db"
wait_for_container_to_become_ready "$ACSMS_NAMESPACE" "io.kompose.service=db"
log "Database is ready."

# Deploy MS components.
apply "${MANIFESTS_DIR}/fleet-manager"
wait_for_container_to_appear "$ACSMS_NAMESPACE" "io.kompose.service=fleet-manager" "db-migrate"
wait_for_container_to_appear "$ACSMS_NAMESPACE" "io.kompose.service=fleet-manager" "fleet-manager"
if [[ "$SPAWN_LOGGER" == "true" ]]; then
    $KUBECTL -n "$ACSMS_NAMESPACE" logs -l io.kompose.service=fleet-manager --all-containers --pod-running-timeout=1m --since=1m --tail=100 -f >"${LOG_DIR}/pod-logs_fleet-manager.txt" 2>&1 &
fi

apply "${MANIFESTS_DIR}/fleetshard-sync"
wait_for_container_to_appear "$ACSMS_NAMESPACE" "io.kompose.service=fleetshard-sync" "fleetshard-sync"
if [[ "$SPAWN_LOGGER" == "true" ]]; then
    $KUBECTL -n "$ACSMS_NAMESPACE" logs -l io.kompose.service=fleetshard-sync --all-containers --pod-running-timeout=1m --since=1m --tail=100 -f >"${LOG_DIR}/pod-logs_fleetshard-sync_fleetshard-sync.txt" 2>&1 &
fi

# Prerequisite for port-forwarding are pods in ready state.
wait_for_container_to_become_ready "$ACSMS_NAMESPACE" "io.kompose.service=fleet-manager"

if [[ "$ENABLE_FM_PORT_FORWARDING" == "true" ]]; then
    port-forwarding start fleet-manager 8000 8000
fi

if [[ "$ENABLE_DB_PORT_FORWARDING" == "true" ]]; then
    port-forwarding start db 5432 5432
fi

log "** Fleet Manager ready ** "
