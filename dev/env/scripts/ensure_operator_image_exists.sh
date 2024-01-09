#!/usr/bin/env bash
set -eo pipefail

operator_images=$(yq -o json -M dev/config/gitops-config.yaml | jq -r '.rhacsOperators.operators[].image')
operator_images="${operator_images} registry.redhat.io/openshift4/ose-kube-rbac-proxy:v4.13" # keep in sync with fleetshard/pkg/central/charts/data/rhacs-operator/templates/rhacs-operator-deployment.yaml

for operator_image in $operator_images; do
    if [[ -n "$(docker images -q "$operator_image")" ]]; then
      echo "Found image $operator_image locally"
      continue
    fi
    docker pull "$operator_image"
    if [[ "$CLUSTER_TYPE" == "kind" ]]; then
        kind load docker-image "$operator_image"
    fi
done
