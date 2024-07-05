#!/usr/bin/env bash

# The initial script executed for openshift/release CI jobs.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
# shellcheck source=scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"

# This file is used to fit into the ocp 4 infra cluster workflow defined in:
# https://github.com/openshift/release/blob/master/ci-operator/step-registry/stackrox/automation-flavors/ocp-4-e2e/stackrox-automation-flavors-ocp-4-e2e-workflow.yaml

info "Running stackrox OSCI workflow"
info "Setting worker node type and count for OCP 4 jobs"
set_ci_shared_export WORKER_NODE_COUNT 2
set_ci_shared_export WORKER_NODE_TYPE e2-standard-8
