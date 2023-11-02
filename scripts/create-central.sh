#!/usr/bin/env bash
set -eo pipefail
name=${1:-"test-central-1"}

echo "Creating central tenant: $name"


# shellcheck disable=SC1001
curl -X POST -H "Authorization: Bearer $(ocm token)" -H "Content-Type: application/json" \
  http://127.0.0.1:8000/api/rhacs/v1/centrals\?async\=true \
  -d '{"name": "'${name}'", "multi_az": true, "cloud_provider": "standalone", "region": "standalone"}'
