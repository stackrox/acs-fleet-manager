#!/usr/bin/env bash

set -eu -o pipefail

GITROOT="$(git rev-parse --show-toplevel)"
export GITROOT
# shellcheck source=/dev/null
source "${GITROOT}/dev/env/scripts/lib.sh"

export DOCKER=${DOCKER:-docker}
export OC=${OC:-oc}
OPENSHIFT_CI=${OPENSHIFT_CI:-false}
DRY_RUN=${DRY_RUN:-"false"}
FLEET_MANAGER_IMAGE=${FLEET_MANAGER_IMAGE:-}

if [[ "$OPENSHIFT_CI" == "true" ]]; then
    vault_name="rhacs-ms-push"
    vault_mount="/var/run/${vault_name}"
    log "Retrieving secrets from Vault mount ${vault_mount}"
    shopt -s nullglob
    for cred in "${vault_mount}/"[A-Z]*; do
        secret_name="$(basename "$cred")"
        secret_value="$(cat "$cred")"
        log "Got secret ${secret_name}"
        export "${secret_name}"="${secret_value}"
    done

    IMAGE_PUSH_REGISTRY="quay.io/rhacs-eng"
    IMAGE_PUSH_USERNAME=${QUAY_RHACS_ENG_RW_USERNAME:-}
    IMAGE_PUSH_PASSWORD=${QUAY_RHACS_ENG_RW_PASSWORD:-}
    DOCKER_CONFIG="${HOME}/.docker"

    if [[ -z "$IMAGE_PUSH_USERNAME" ]]; then
        die "Error: Could not find secret QUAY_RHACS_ENG_RW_USERNAME in CI Vault ${vault_name}"
    fi

    if [[ -z "$IMAGE_PUSH_PASSWORD" ]]; then
        die "Error: Could not find secret QUAY_RHACS_ENG_RW_PASSWORD in CI Vault ${vault_name}"
    fi
else
    IMAGE_PUSH_REGISTRY=${IMAGE_PUSH_REGISTRY:-}
    IMAGE_PUSH_USERNAME=${IMAGE_PUSH_USERNAME:-}
    IMAGE_PUSH_PASSWORD=${IMAGE_PUSH_PASSWORD:-}
    DOCKER_CONFIG="${PWD}/.docker"

    if [[ -z "$IMAGE_PUSH_REGISTRY" ]]; then
        die "Error: IMAGE_PUSH_REGISTRY not found in the environment."
    fi

    if [[ -z "$IMAGE_PUSH_USERNAME" ]]; then
        die "Error: IMAGE_PUSH_USERNAME not found in the environment."
    fi

    if [[ -z "$IMAGE_PUSH_PASSWORD" ]]; then
        die "Error: IMAGE_PUSH_PASSWORD not found in the environment."
    fi
fi

log
log "** Entrypoint for ACS MS Image Push **"
log

registry_host=$(echo "$IMAGE_PUSH_REGISTRY" | cut -d / -f 1)
tag=$(make -s -C "$GITROOT" tag)
image_tag="${IMAGE_PUSH_REGISTRY}/acs-fleet-manager:${tag}"

if [[ -z "$IMAGE_PUSH_REGISTRY" ]]; then
    die "Error: IMAGE_PUSH_REGISTRY not found."
fi

if [[ -z "$FLEET_MANAGER_IMAGE" ]]; then
    die "Error: FLEET_MANAGER_IMAGE not found."
fi

if [[ "$tag" =~ dirty$ ]]; then
    die "Error: Repository is dirty, refusing to push dirty tag to registry."
fi

log "Image:        ${FLEET_MANAGER_IMAGE}"
log "Version:      ${tag}"
log "Tag:          ${image_tag}"
log "Registry:     ${IMAGE_PUSH_REGISTRY}"
log "OpenShift CI: ${OPENSHIFT_CI}"
log

docker_logged_in() {
    local host="$1"
    local cfg=""
    if [[ -f "${DOCKER_CONFIG}/config.json" ]]; then
        cfg=$(cat "${DOCKER_CONFIG}/config.json")
    fi
    if echo "$cfg" | jq -r ".auths | keys[]" | grep -q "^${host}$"; then
        return 0
    else
        return 1
    fi
}

if [[ "$OPENSHIFT_CI" == "true" ]]; then
    tmp_docker_config="/tmp/config.json"
    if [[ -f "${DOCKER_CONFIG}/config.json" ]]; then
        cp "${DOCKER_CONFIG}/config.json" /tmp/config.json
    else
        touch "$tmp_docker_config"
    fi
    log "Logging into build cluster registry..."
    oc registry login --to "$tmp_docker_config"
    log "Logging into Quay..."
    oc registry login --auth-basic="${IMAGE_PUSH_USERNAME}:${IMAGE_PUSH_PASSWORD}" --registry="$registry_host" --to "$tmp_docker_config"
    log "Mirroring ${FLEET_MANAGER_IMAGE} to ${image_tag}..."
    oc image mirror "$FLEET_MANAGER_IMAGE" "$image_tag" -a "$tmp_docker_config"
else
    if ! docker_logged_in "$registry_host"; then
        log "Logging into ${registry_host}..."
        $DOCKER login -u "$IMAGE_PUSH_USERNAME" --password-stdin "$registry_host" <<<"$IMAGE_PUSH_PASSWORD"
        log "Done"
    fi

    log "Creating tag ${image_tag}..."
    $DOCKER tag "$FLEET_MANAGER_IMAGE" "$image_tag"
    log "Done"

    if [[ "$DRY_RUN" == "false" ]]; then
        log "Pushing tag ${image_tag}"
        $DOCKER push "$image_tag"
        log "Done"
    else
        log "Skipping push because DRY_RUN is set to '$DRY_RUN'"
        log "Would have executed: $DOCKER push \"${image_tag}\""
    fi
fi
