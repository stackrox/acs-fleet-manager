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
