#!/usr/bin/env bash
set -eo pipefail

SCRIPT="$(python3 -c 'import os, sys; print(os.path.realpath(sys.argv[1]))' "${BASH_SOURCE[0]}")"
DIRNAME=$(dirname "$SCRIPT")

echo "Copying cluster configuration"
cp "$DIRNAME/../config/dataplane-cluster-configuration-minikube.yaml" "$DIRNAME/../../config/dataplane-cluster-configuration.yaml"

# TODO: create Central request
# ./"$DIRNAME"/create-central.sh
