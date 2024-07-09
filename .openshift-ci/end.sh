#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
# shellcheck source=scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"

# The initial script executed for openshift/release CI jobs.
set -euo pipefail

# As of now this file is only a placeholder to make the repo fit to the
# workflow defined for ocp infra clusters in openshift/release repository
# https://github.com/openshift/release/blob/master/ci-operator/step-registry/stackrox/automation-flavors/ocp-4-e2e/stackrox-automation-flavors-ocp-4-e2e-workflow.yaml
log "End of stackrox OSCI workflow"
