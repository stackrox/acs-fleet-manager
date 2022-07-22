#!/usr/bin/env bash

set -eu -o pipefail

GITROOT="$(git rev-parse --show-toplevel)"
export GITROOT
# shellcheck source=/dev/null
source "${GITROOT}/dev/env/scripts/lib.sh"

if [[ "${OPENSHIFT_CI:-false}" != "true" ]]; then
    die "Error: This script must be executed from OpenShift CI"
fi

IMAGE_PUSH_REGISTRY="quay.io/rhacs-eng"
VAULT_NAME="rhacs-ms-push"
VAULT_MOUNT="/var/run/${VAULT_NAME}"
FLEET_MANAGER_IMAGE=${FLEET_MANAGER_IMAGE:-}

log "Retrieving secrets from Vault mount ${VAULT_MOUNT}"
shopt -s nullglob
for cred in "${VAULT_MOUNT}/"[A-Z]*; do
    secret_name="$(basename "$cred")"
    secret_value="$(cat "$cred")"
    log "Retrieved secret ${secret_name}"
    export "${secret_name}"="${secret_value}"
done

QUAY_RHACS_ENG_RW_USERNAME=${QUAY_RHACS_ENG_RW_USERNAME:-}
QUAY_RHACS_ENG_RW_PASSWORD=${QUAY_RHACS_ENG_RW_PASSWORD:-}

if [[ -z "$FLEET_MANAGER_IMAGE" ]]; then
    die "Error: FLEET_MANAGER_IMAGE not found."
fi

if [[ -z "$QUAY_RHACS_ENG_RW_USERNAME" ]]; then
    die "Error: Could not find secret QUAY_RHACS_ENG_RW_USERNAME in CI Vault ${VAULT_NAME}"
fi

if [[ -z "$QUAY_RHACS_ENG_RW_PASSWORD" ]]; then
    die "Error: Could not find secret QUAY_RHACS_ENG_RW_PASSWORD in CI Vault ${VAULT_NAME}"
fi

log
log "** Entrypoint for ACS MS Image Push **"
log

registry_host=$(echo "$IMAGE_PUSH_REGISTRY" | cut -d / -f 1)
tag=$(make -s -C "$GITROOT" tag)
image_tag="${IMAGE_PUSH_REGISTRY}/fleet-manager:${tag}"

if [[ "$tag" =~ dirty$ ]]; then
    die "Error: Repository is dirty, refusing to push dirty tag to registry."
fi

log "Image:        ${FLEET_MANAGER_IMAGE}"
log "Version:      ${tag}"
log "Tag:          ${image_tag}"
log "Registry:     ${IMAGE_PUSH_REGISTRY}"
log

TMP_DOCKER_CONFIG="/tmp/config.json"
touch "$TMP_DOCKER_CONFIG"

log "Logging into build cluster registry..."
oc registry login --to "$TMP_DOCKER_CONFIG"
log "Logging into Quay..."
oc registry login --auth-basic="${QUAY_RHACS_ENG_RW_USERNAME}:${QUAY_RHACS_ENG_RW_PASSWORD}" --registry="$registry_host" --to "$TMP_DOCKER_CONFIG"
log "Mirroring ${FLEET_MANAGER_IMAGE} to ${image_tag}..."
oc image mirror "$FLEET_MANAGER_IMAGE" "$image_tag" -a "$TMP_DOCKER_CONFIG"
