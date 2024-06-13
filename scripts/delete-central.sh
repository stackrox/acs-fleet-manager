#!/usr/bin/env bash
set -eo pipefail
id=${1}

if [ -z "$id" ]; then
  echo "Usage: $0 <central_id>"
  exit 1
fi

echo "Deleting central $id"


# shellcheck disable=SC1001
curl -X DELETE -H "Authorization: Bearer $(ocm token)" \
  http://127.0.0.1:8000/api/rhacs/v1/centrals/${id}\?async\=true
