#!/usr/bin/env bash

# The initial script executed for openshift/release CI jobs.
set -euo pipefail

# This file is used to fit into the ocp 4 infra cluster workflow defined in:
# https://github.com/openshift/release/blob/master/ci-operator/step-registry/stackrox/automation-flavors/ocp-4-e2e/stackrox-automation-flavors-ocp-4-e2e-workflow.yaml

# set_ci_shared_export() - for openshift-ci this is state shared between steps.
set_ci_shared_export() {
    if [[ "$#" -ne 2 ]]; then
        die "missing args. usage: set_ci_shared_export <env-name> <env-value>"
    fi

    ci_export "$@"

    local env_name="$1"
    local env_value="$2"

    echo "export ${env_name}=${env_value}" | tee -a "${SHARED_DIR:-/tmp}/shared_env"
}

info "Running stackrox OSCI workflow"
info "Setting worker node type and count for OCP 4 jobs"
set_ci_shared_export WORKER_NODE_COUNT 2
set_ci_shared_export WORKER_NODE_TYPE e2-standard-8
