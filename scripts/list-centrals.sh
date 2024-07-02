#!/usr/bin/env bash
set -eo pipefail

# shellcheck disable=SC1001
curl -X GET -H "Authorization: Bearer $(ocm token)" \
  http://127.0.0.1:8000/api/rhacs/v1/centrals/\?async\=true
