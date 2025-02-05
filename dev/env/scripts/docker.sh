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

ensure_fleet_manager_image_exists() {
    if should_skip_image_build;  then
        return
    fi
    if $DOCKER image inspect "$FLEET_MANAGER_IMAGE" >/dev/null 2>&1; then
        log "Image ${FLEET_MANAGER_IMAGE} found, skipping building of a new image."
        return
    fi
    if [[ "$FLEET_MANAGER_IMAGE" != "$(make -s -C "${GITROOT}" full-image-tag)" ]]; then
        die "Cannot find image '${FLEET_MANAGER_IMAGE}' and don't know how to build it"
    fi
    if [[ "$CLUSTER_TYPE" == "infra-openshift" ]]; then
        log "Building local image and pushing it to the internal registry"
        make -C "${GITROOT}" image/push/internal

        # Override image tag from an image stream reference because image streams are not compatible with Helm 3 and image lookup can't be used for fleetshard-sync deployment
        FLEET_MANAGER_IMAGE=$(oc get istag/fleet-manager:"$(make -s -C "${GITROOT}" tag)" -n "${ACSCS_NAMESPACE}" -o jsonpath='{.image.dockerImageReference}')
    else
        log "Building local image..."
        make -C "${GITROOT}" image/build
        if [[ "${CLUSTER_TYPE}" == "kind" ]]; then
            kind load docker-image "$FLEET_MANAGER_IMAGE"
        fi
        if [[ "${CLUSTER_TYPE}" == "crc" ]]; then
            $DOCKER tag "$FLEET_MANAGER_IMAGE" "${ACSCS_NAMESPACE}/$FLEET_MANAGER_IMAGE"
        fi
    fi
}

ensure_fleetshard_operator_image_exists() {
    if should_skip_image_build; then
        if [[ -z "${FLEETSHARD_OPERATOR_IMAGE:-}" ]]; then
            die "FLEET_MANAGER_IMAGE is not set"
        fi
        return
    fi

    if [[ -z "${FLEETSHARD_OPERATOR_IMAGE:-}" ]]; then
        FLEETSHARD_OPERATOR_IMAGE="fleetshard-operator:$(make tag)"
        export FLEETSHARD_OPERATOR_IMAGE
        if [[ "$CLUSTER_TYPE" == "infra-openshift" ]]; then
            log "Building fleetshard operator image ${FLEETSHARD_OPERATOR_IMAGE} and pushing it to internal registry"
            make -C "${GITROOT}" image/push/fleetshard-operator/internal
        else
            log "Building fleetshard operator image ${FLEETSHARD_OPERATOR_IMAGE}..."
            make -C "${GITROOT}" image/build/fleetshard-operator IMAGE_REF="${FLEETSHARD_OPERATOR_IMAGE}"
            if [[ "${CLUSTER_TYPE}" == "kind" ]]; then
                kind load docker-image "$FLEETSHARD_OPERATOR_IMAGE"
            fi
            if [[ "${CLUSTER_TYPE}" == "crc" ]]; then
                $DOCKER tag "$FLEETSHARD_OPERATOR_IMAGE" "${ACSCS_NAMESPACE}/$FLEETSHARD_OPERATOR_IMAGE"
            fi
        fi
    fi
}

should_skip_image_build() {
    if [[ "$CLUSTER_TYPE" == "openshift-ci" ]]; then
        return 0
    fi
    if is_running_inside_docker; then
        return 0
    fi
    return 1
}
