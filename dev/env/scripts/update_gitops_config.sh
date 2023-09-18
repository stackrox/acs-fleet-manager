#!/bin/bash

set -eu -o pipefail

GITROOT="$(git rev-parse --show-toplevel)"
source "${GITROOT}/dev/env/scripts/lib.sh"
init

apply "$GITROOT/dev/env/manifests/fleet-manager/04-gitops-config.yaml"
