#!/bin/bash
set -eo pipefail

GITROOT="$(git rev-parse --show-toplevel)"
export GITROOT

export DOCKER=${DOCKER:-docker}
export CLUSTER_TYPE=${CLUSTER_TYPE:-default}
export AWS_AUTH_HELPER=aws-saml

# shellcheck source=dev/env/scripts/docker.sh
source "${GITROOT}/dev/env/scripts/docker.sh"
# shellcheck source=scripts/lib/external_config.sh
source "${GITROOT}/scripts/lib/external_config.sh"

init_chamber

AWS_ACCESS_KEY_ID=$(aws configure get aws_access_key_id --profile=saml)
AWS_SECRET_ACCESS_KEY=$(aws configure get aws_access_key_id --profile=saml)
AWS_SESSION_TOKEN=$(aws configure get aws_session_token --profile=saml)

FLEET_MANAGER_IMAGE=$(make -s -C "$GITROOT" full-image-tag)

# Run the necessary docker actions out of the container
ensure_fleet_manager_image_exists

docker build -t acscs-e2e -f "$GITROOT/.openshift-ci/e2e-runtime/Dockerfile" "${GITROOT}"

docker run \
    -v "${KUBECONFIG:-$HOME/.kube/config}":/var/kubeconfig -e KUBECONFIG=/var/kubeconfig \
    -e STATIC_TOKEN="$STATIC_TOKEN" -e STATIC_TOKEN_ADMIN="$STATIC_TOKEN_ADMIN" \
    -e QUAY_USER="$QUAY_USER" -e QUAY_TOKEN="$QUAY_TOKEN" \
    -e AWS_AUTH_HELPER=none -e AWS_SESSION_TOKEN="$AWS_SESSION_TOKEN" \
    -e AWS_ACCESS_KEY_ID="$AWS_ACCESS_KEY_ID" -e AWS_SECRET_ACCESS_KEY="$AWS_SECRET_ACCESS_KEY" \
    -e FLEET_MANAGER_IMAGE="$FLEET_MANAGER_IMAGE" \
    --net=host --name acscs-e2e --rm acscs-e2e
