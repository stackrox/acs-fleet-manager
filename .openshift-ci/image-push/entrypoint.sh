#!/usr/bin/env bash

export GITROOT="$(git rev-parse --show-toplevel)"
source "${GITROOT}/dev/env/scripts/lib.sh"
init

log
log "** Entrypoint for ACS MS Image Push **"
log

IMAGE_PUSH_REGISTRY=${IMAGE_PUSH_REGISTRY:-}
IMAGE_PUSH_USERNAME=${IMAGE_PUSH_USERNAME:-}
IMAGE_PUSH_PASSWORD=${IMAGE_PUSH_PASSWORD:-}
OPENSHIFT_CI=${OPENSHIFT_CI:-false}

if [[ "$OPENSHIFT_CI" == "true" ]]; then
    log "Retrieving secrets from Vault mount"
    shopt -s nullglob
    for cred in /var/run/rhacs-ms-e2e-tests/[A-Z]*; do
        secret_name="$(basename "$cred")"
        secret_value="$(cat "$cred")"
        log "Got secret ${secret_name}"
        export "${secret_name}"="${secret_value}"
    done
fi

if [[ -z "$IMAGE_PUSH_REGISTRY" ]]; then
    die "Error: IMAGE_PUSH_REGISTRY not found. Either set it in the environment or make sure that the CI Vault contains this data."
fi

if [[ -z "$FLEET_MANAGER_IMAGE" ]]; then
    die "Error: FLEET_MANAGER_IMAGE not provided by CI Operator configuration.\nDon't know which image to push."
fi

registry_host=$(echo "$IMAGE_PUSH_REGISTRY" | cut -d / -f 1)
tag=$(make -s -C "$GITROOT" tag)
image_tag="${IMAGE_PUSH_REGISTRY}/acsms-test:${tag}"

log "Image:   ${FLEET_MANAGER_IMAGE}"
log "Version: ${tag}"
log "Tag:     ${image_tag}"

if [[ "$tag" =~ dirty$ ]]; then
    die "Error: Repository is dirty, refusing to push dirty tag to registry."
fi

if [[ "$OPENSHIFT_CI" == "true" ]]; then
    if [[ -z "$IMAGE_PUSH_USERNAME" ]]; then
        die "Error: IMAGE_PUSH_USERNAME not found. Either set it in the environment or make sure that the CI Vault contains this data."
    fi

    if [[ -z "$IMAGE_PUSH_PASSWORD" ]]; then
        die "Error: IMAGE_PUSH_PASSWORD not found. Either set it in the environment or make sure that the CI Vault contains this data."
    fi

    if [[ -f ~/.docker/config.json ]]; then
        cp ~/.docker/config.json /tmp/config.json
    else
        touch /tmp/config.json
    fi
    oc registry login --to /tmp/config.json
    oc registry login --auth-basic="${IMAGE_PUSH_USERNAME}:${IMAGE_PUSH_PASSWORD}" --registry="$registry_host" --to /tmp/config.json
    log "Mirroring ${FLEET_MANAGER_IMAGE} to ${image_tag}"
    oc image mirror "$FLEET_MANAGER_IMAGE" "$image_tag" -a /tmp/config.json
else
    docker_logged_in() {
        local host="$1"
        local cfg=""
        local docker_cfg="$HOME/.docker/config.json"
        if [[ -f "$docker_cfg" ]]; then
            cfg=$(cat "$docker_cfg")
        fi
        if echo "$cfg" | jq -r ".auths | keys[]" | grep -q "^${host}$"; then
            return 0
        else
            return 1
        fi
    }

    if ! docker_logged_in "$registry_host"; then
        if [[ -z "$IMAGE_PUSH_USERNAME" ]]; then
            die "Error: IMAGE_PUSH_USERNAME not found. Either set it in the environment or make sure that the CI Vault contains this data."
        fi

        if [[ -z "$IMAGE_PUSH_PASSWORD" ]]; then
            die "Error: IMAGE_PUSH_PASSWORD not found. Either set it in the environment or make sure that the CI Vault contains this data."
        fi

        log "Logging into ${registry_host}."
        docker login -u "$IMAGE_PUSH_USERNAME" --password-stdin "$registry_host" <<<"$IMAGE_PUSH_PASSWORD"
    fi

    log "Creating tag ${image_tag}"
    docker tag "$FLEET_MANAGER_IMAGE" "$image_tag"

    log "Pushing tag ${image_tag}"
    docker push "$image_tag"
fi
