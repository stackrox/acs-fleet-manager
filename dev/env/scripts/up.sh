#!/usr/bin/env bash

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
    $MINIKUBE ssh 'docker rm --force $(docker ps -a -q) > /dev/null' || true
    sleep 1
    $MINIKUBE image ls | grep -v "^${FLEET_MANAGER_IMAGE}$" | { grep "^.*/fleet-manager-.*/fleet-manager:.*$" || test $? = 1; } | while read img; do
        $MINIKUBE image rm "$img"
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
for i in $(seq 10); do
    if $KUBECTL -n "$ACSMS_NAMESPACE" wait --timeout=5s --for=condition=ready pod -l io.kompose.service=db 2>/dev/null >&2; then
        break
    else
        sleep 1
    fi
done

# Deploy MS components.
apply "${MANIFESTS_DIR}/fleet-manager"
if [[ "$SPAWN_LOGGER" == "true" ]]; then
    # Wait for init Container to be in running or in terminated state:
    for i in $(seq 5); do
        state=$({
            $KUBECTL -n "$ACSMS_NAMESPACE" get pod -l io.kompose.service=fleet-manager -o jsonpath='{.items[0].status.initContainerStatuses[0].state}'
            echo '{}'
        } |
            jq -r 'keys[]')
        echo "state = $state"
        if [[ "$state" == "terminated" || "$state" == "running" ]]; then
            break
        fi
        sleep 1
    done
    $KUBECTL -n "$ACSMS_NAMESPACE" logs -l io.kompose.service=fleet-manager --all-containers --pod-running-timeout=1m --since=1m --tail=100 -f >"${LOG_DIR}/pod-logs_fleet-manager.txt" 2>&1 &
fi

apply "${MANIFESTS_DIR}/fleetshard-sync"
if [[ "$SPAWN_LOGGER" == "true" ]]; then
    # Wait for init Container to be in running or in terminated state:
    for i in $(seq 5); do
        state=$({
            $KUBECTL -n "$ACSMS_NAMESPACE" get pod -l io.kompose.service=fleetshard-sync -o jsonpath='{.items[0].status.containerStatuses[0].state}'
            echo '{}'
        } |
            jq -r 'keys[]')
        if [[ "$state" == "terminated" || "$state" == "running" ]]; then
            break
        fi
        sleep 1
    done
    $KUBECTL -n "$ACSMS_NAMESPACE" logs -l io.kompose.service=fleetshard-sync --all-containers --pod-running-timeout=1m --since=1m --tail=100 -f >"${LOG_DIR}/pod-logs_fleetshard-sync_fleetshard-sync.txt" 2>&1 &
fi

# Prerequisite for port-forwarding are pods in ready state.
$KUBECTL -n "$ACSMS_NAMESPACE" wait --timeout=5s --for=condition=ready pod -l io.kompose.service=db
for i in $(seq 10); do
    if $KUBECTL -n "$ACSMS_NAMESPACE" wait --timeout=5s --for=condition=ready pod -l io.kompose.service=fleet-manager 2>/dev/null >&2; then
        break
    else
        sleep 1
    fi
done
$KUBECTL -n "$ACSMS_NAMESPACE" wait --timeout=120s --for=condition=ready pod -l io.kompose.service=fleet-manager
sleep 1

if [[ "$ENABLE_FM_PORT_FORWARDING" == "true" ]]; then
    log "Setting up port-forwarding: fleet-manager is at http://localhost:8000"
    enable-port-forwarding start fleet-manager 8000 8000
fi

if [[ "$ENABLE_DB_PORT_FORWARDING" == "true" ]]; then
    log "Setting up port-forwarding: db is at localhost:5432"
    enable-port-forwarding start db 5432 5432
fi

log "** Fleet Manager ready ** "
