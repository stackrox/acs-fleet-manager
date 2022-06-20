#!/usr/bin/env bash
set -eo pipefail

echo "Creating central tenant: test-central-1"

curl -X POST -H "Authorization: Bearer $(ocm token)" -H "Content-Type: application/json" \
  http://127.0.0.1:8000/api/rhacs/v1/centrals\?async\=true \
  -d '{"name": "test-central-1", "multi_az": true, "cloud_provider": "standalone", "region": "standalone"}'
