#!/usr/bin/env bash

docker run \
    -v "${KUBECONFIG:-$HOME/.kube/config}":/var/kubeconfig -e KUBECONFIG=/var/kubeconfig \
    -e STATIC_TOKEN="$STATIC_TOKEN" -e STATIC_TOKEN_ADMIN="$STATIC_TOKEN_ADMIN" \
    -e QUAY_USER="$QUAY_USER" -e QUAY_TOKEN="$QUAY_TOKEN" \
    -e AWS_AUTH_HELPER=none -e AWS_SESSION_TOKEN="$AWS_SESSION_TOKEN" \
    -e AWS_ACCESS_KEY_ID="$AWS_ACCESS_KEY_ID" -e AWS_SECRET_ACCESS_KEY="$AWS_SECRET_ACCESS_KEY" \
    -e FLEET_MANAGER_IMAGE="$FLEET_MANAGER_IMAGE" \
    --net=host --name acscs-e2e --rm acscs-e2e
