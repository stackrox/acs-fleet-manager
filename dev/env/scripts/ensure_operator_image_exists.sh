#!/usr/bin/env bash
set -eo pipefail

operator_images=$(yq e '.data."config.yaml"' dev/env/manifests/fleet-manager/04-gitops-config.yaml | yq -o json -M | jq -r '.rhacsOperators.operators[].image')

for operator_image in $operator_images; do
    docker pull $operator_image
    if [[ "$CLUSTER_TYPE" == "kind" ]]; then
            kind load docker-image $operator_image
    fi
done
