#!/usr/bin/env bash

GITROOT_DEFAULT=$(git rev-parse --show-toplevel)
export GITROOT=${GITROOT:-$GITROOT_DEFAULT}

# shellcheck source=/dev/null
source "$GITROOT/scripts/lib/log.sh"

_docker_images=""

is_running_inside_docker() {
    if [[ -f "/.dockerenv" ]]; then
        return 0
    fi
    return 1
}

docker_pull() {
    local image_ref="${1:-}"
    if [[ -z "${_docker_images}" ]]; then
        _docker_images=$($DOCKER images --format '{{.Repository}}:{{.Tag}}')
    fi
    if echo "${_docker_images}" | grep -q "^${image_ref}$"; then
        log "Skipping pulling of image ${image_ref}, as it is already there"
    else
        log "Pulling image ${image_ref}"
        $DOCKER pull "$image_ref"
    fi
    if [[ "$CLUSTER_TYPE" == "kind" ]]; then
        log "Load image $image_ref to kind"
        $KIND load docker-image "$image_ref"
    fi
}

docker_logged_in() {
    local registry="${1:-}"
    if [[ -z "$registry" ]]; then
        log "docker_logged_in() called with empty registry argument"
        return 1
    fi
    if jq -ec ".auths[\"${registry}\"]" <"$DOCKER_CONFIG/config.json" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

ensure_fleet_manager_image_exists() {
    if [[ "$FLEET_MANAGER_IMAGE" =~ ^[0-9a-z.-]+$ ]]; then
        log "FLEET_MANAGER_IMAGE='${FLEET_MANAGER_IMAGE}' looks like an image tag. Setting:"
        FLEET_MANAGER_IMAGE="quay.io/rhacs-eng/fleet-manager:${FLEET_MANAGER_IMAGE}"
        log "FLEET_MANAGER_IMAGE='${FLEET_MANAGER_IMAGE}'"
    fi

    if is_local_deploy; then
        # We are deploying locally. Locally we support Quay images and freshly built images.
        if [[ "$FLEET_MANAGER_IMAGE" =~ ^fleet-manager.*:.* ]]; then
            # Local image reference, which cannot be pulled.
            image_available=$(if $DOCKER image inspect "${FLEET_MANAGER_IMAGE}" >/dev/null 2>&1; then echo "true"; else echo "false"; fi)
            if [[ "$image_available" != "true" || "$FLEET_MANAGER_IMAGE" =~ dirty$ ]]; then
                # Attempt to build this image.
                if [[ "$FLEET_MANAGER_IMAGE" == "$(make -s -C "${GITROOT}" full-image-tag)" ]]; then
                    log "Building local image..."
                    make -C "${GITROOT}" image/build
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

        if [[ "${CLUSTER_TYPE}" == "kind" ]]; then
            kind load docker-image "$FLEET_MANAGER_IMAGE"
        fi
        if [[ "${CLUSTER_TYPE}" == "crc" ]]; then
            docker tag "$FLEET_MANAGER_IMAGE" "${ACSCS_NAMESPACE}/$FLEET_MANAGER_IMAGE"
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
}

ensure_fleetshard_operator_image_exists() {
    if ! is_local_deploy; then
        if [[ -z "${FLEETSHARD_OPERATOR_IMAGE:-}" ]]; then
            die "FLEET_MANAGER_IMAGE is not set"
        fi
        return
    fi

    if [[ -z "${FLEETSHARD_OPERATOR_IMAGE:-}" ]]; then
        FLEETSHARD_OPERATOR_IMAGE="fleetshard-operator:$(make tag)"
        export FLEETSHARD_OPERATOR_IMAGE
        log "Building fleetshard operator image ${FLEETSHARD_OPERATOR_IMAGE}..."
        make -C "${GITROOT}" image/build/fleetshard-operator IMAGE_REF="${FLEETSHARD_OPERATOR_IMAGE}"
        if [[ "${CLUSTER_TYPE}" == "kind" ]]; then
            kind load docker-image "$FLEETSHARD_OPERATOR_IMAGE"
        fi
        if [[ "${CLUSTER_TYPE}" == "crc" ]]; then
            docker tag "$FLEETSHARD_OPERATOR_IMAGE" "${ACSCS_NAMESPACE}/$FLEETSHARD_OPERATOR_IMAGE"
        fi
    fi
}

is_local_deploy() {
    if [[ "$CLUSTER_TYPE" == "openshift-ci" \
        || "$CLUSTER_TYPE" == "infra-openshift" \
        || "$CLUSTER_TYPE" == "gke" \
        ]]; then
        return 1
    fi
    if is_running_inside_docker; then
        return 1
    fi
    return 0
}
