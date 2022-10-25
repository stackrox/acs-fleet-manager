#!/usr/bin/env bash

## This script takes care of deploying Managed Service components.

GITROOT="$(git rev-parse --show-toplevel)"
export GITROOT
# shellcheck source=/dev/null
source "${GITROOT}/dev/env/scripts/lib.sh"
# shellcheck source=/dev/null
source "${GITROOT}/scripts/lib/external_config.sh"

init
init_chamber

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
Use AWS Vault: ${USE_AWS_VAULT}
Debugging Mode: ${DEBUG_PODS}

EOF

KUBE_CONFIG=$(assemble_kubeconfig | yq e . -o=json - | jq -c . -)
export KUBE_CONFIG

if [[ "$FLEET_MANAGER_IMAGE" =~ ^[0-9a-z.-]+$ ]]; then
    log "FLEET_MANAGER_IMAGE='${FLEET_MANAGER_IMAGE}' looks like an image tag. Setting:"
    FLEET_MANAGER_IMAGE="quay.io/rhacs-eng/fleet-manager:${FLEET_MANAGER_IMAGE}"
    log "FLEET_MANAGER_IMAGE='${FLEET_MANAGER_IMAGE}'"
fi

if [[ ! ("$CLUSTER_TYPE" == "openshift-ci" || "$CLUSTER_TYPE" == "infra-openshift") ]]; then
    # We are deploying locally. Locally we support Quay images and freshly built images.
    if [[ "$FLEET_MANAGER_IMAGE" =~ ^fleet-manager.*:.* ]]; then
        # Local image reference, which cannot be pulled.
        image_available=$(if $DOCKER image inspect "${FLEET_MANAGER_IMAGE}" >/dev/null 2>&1; then echo "true"; else echo "false"; fi)
        if [[ "$image_available" != "true" || "$FLEET_MANAGER_IMAGE" =~ dirty$ ]]; then
            # Attempt to build this image.
            if [[ "$FLEET_MANAGER_IMAGE" == "$(make -s -C "${GITROOT}" full-image-tag)" ]]; then
                # Looks like we can build this tag from the current state of the repository.
                if [[ "$DEBUG_PODS" == "true" ]]; then
                    log "Building image with debugging support..."
                    make -C "${GITROOT}" image/build/multi-target
                else
                    # We *could* also use image/build/multi-target, because that
                    # target also supports building of standard (i.e. non-debug) images.
                    # But until there is a reliable and portable caching mechanism for dockerized
                    # Go projects, this would be regression in terms of build performance.
                    # Hence we don't use the image/build/multi-target target here, but the
                    # older `image/build/local` target, which uses a hybrid building
                    # approach and is much faster.
                    log "Building standard image..."
                    make -C "${GITROOT}" image/build/local
                fi
            else
                die "Cannot find image '${FLEET_MANAGER_IMAGE}' and don't know how to build it"
            fi
        else
            log "Image ${FLEET_MANAGER_IMAGE} found, skipping building of a new image."
        fi
    else
        log "Trying to pull image '${FLEET_MANAGER_IMAGE}'..."
        docker_pull "$FLEET_MANAGER_IMAGE"
    fi

    # Verify that the image is there.
    if ! $DOCKER image inspect "$FLEET_MANAGER_IMAGE" >/dev/null 2>&1; then
        die "Image ${FLEET_MANAGER_IMAGE} not available in cluster, aborting"
    fi
else
    # We are deploying to a remote cluster.
    if [[ "$FLEET_MANAGER_IMAGE" =~ ^fleet-manager:.* ]]; then
        die "Error: When deploying to a remote target cluster FLEET_MANAGER_IMAGE must point to an image pullable from the target cluster."
    fi
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
run_chamber exec "fleet-manager" -- apply "${MANIFESTS_DIR}/fleet-manager"
wait_for_container_to_appear "$ACSMS_NAMESPACE" "application=fleet-manager" "fleet-manager"
if [[ "$SPAWN_LOGGER" == "true" && -n "${LOG_DIR:-}" ]]; then
    $KUBECTL -n "$ACSMS_NAMESPACE" logs -l application=fleet-manager --all-containers --pod-running-timeout=1m --since=1m --tail=100 -f >"${LOG_DIR}/pod-logs_fleet-manager.txt" 2>&1 &
fi

log "Deploying fleetshard-sync"
run_chamber exec "fleetshard-sync" -- apply "${MANIFESTS_DIR}/fleetshard-sync"
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

log
log "** Fleet Manager ready ** "
log
