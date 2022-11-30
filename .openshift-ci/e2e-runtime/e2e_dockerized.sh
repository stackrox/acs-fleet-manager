#!/bin/bash
set -eo pipefail

GITROOT="$(git rev-parse --show-toplevel)"
export GITROOT
# shellcheck source=dev/env/scripts/docker.sh
source "${GITROOT}/dev/env/scripts/docker.sh"
# shellcheck source=scripts/lib/external_config.sh
source "${GITROOT}/scripts/lib/external_config.sh"

init_chamber

FLEET_MANAGER_IMAGE=$(make -s -C "$GITROOT" full-image-tag)
export FLEET_MANAGER_IMAGE

# Run the necessary docker actions out of the container
preload_dependency_images
ensure_fleet_manager_image_exists

docker build -t acscs-e2e -f "$GITROOT/.openshift-ci/e2e-runtime/Dockerfile" "${GITROOT}"

aws-vault exec dev -- "$GITROOT/.openshift-ci/e2e-runtime/docker_run_e2e.sh"
