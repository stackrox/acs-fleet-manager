#!/usr/bin/env bash
set -oe pipefail

SCRIPT="$(python3 -c 'import os, sys; print(os.path.realpath(sys.argv[1]))' "${BASH_SOURCE[0]}")"
DIRNAME=$(dirname "$SCRIPT")

make secrets/touch

set +e
$(docker ps | grep -q fleet-manager-db)
setup_db_container=$?
set -e

if [[ "$setup_db_container" -ne "0" ]]; then
  echo "Setting up fleet-manager database"
  make db/setup
else
  docker start fleet-manager-db
fi

echo "Run fleet-manager migrations"
make db/migrate

echo "Copying cluster configuration"
cp "$DIRNAME/../dev/config/dataplane-cluster-configuration-minikube.yaml" "$DIRNAME/../config/dataplane-cluster-configuration.yaml"

make binary
OCM_TOKEN=$(ocm token) ./fleet-manager serve
