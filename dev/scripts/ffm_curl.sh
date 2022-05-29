#!/usr/bin/env bash
set -eo pipefail

resource=${1}
 shift
 curl -v -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/${resource} $@
