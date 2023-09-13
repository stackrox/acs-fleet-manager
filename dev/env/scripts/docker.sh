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

        if [[ "${CLUSTER_TYPE}" == "kind" ]]; then
            kind load docker-image "$FLEET_MANAGER_IMAGE"
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

is_local_deploy() {
    if [[ "$CLUSTER_TYPE" == "openshift-ci" || "$CLUSTER_TYPE" == "infra-openshift" ]]; then
        return 1
    fi
    if is_running_inside_docker; then
        return 1
    fi
    return 0
}

preload_dependency_images() {
    if is_running_inside_docker; then
        return
    fi
    log "Preloading images into ${CLUSTER_TYPE} cluster..."
    docker_pull "postgres:13"
    if [[ "$INSTALL_OPERATOR" == "true" || "$RHACS_TARGETED_OPERATOR_UPGRADES" == "true" ]]; then
        # Preload images required by Central installation.
        docker_pull "${IMAGE_REGISTRY}/scanner:${SCANNER_VERSION}"
        docker_pull "${IMAGE_REGISTRY}/scanner-db:${SCANNER_VERSION}"
        docker_pull "${IMAGE_REGISTRY}/main:${CENTRAL_VERSION}"
        docker_pull "${IMAGE_REGISTRY}/central-db:${CENTRAL_VERSION}"
    fi
    log "Images preloaded"
}
