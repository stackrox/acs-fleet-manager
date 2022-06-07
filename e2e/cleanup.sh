#!/usr/bin/env bash
set -eo pipefail

kubectl delete ns e2e-test-central
docker stop fleet-manager-db && docker rm fleet-manager-db
