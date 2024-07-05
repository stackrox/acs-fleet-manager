#!/usr/bin/env bash

# The initial script executed for openshift/release CI jobs.
set -euo pipefail

# As of now this file is only a placeholder to make the repo fit to the
# workflow defined for ocp infra clusters in openshift/release repository
# https://github.com/openshift/release/blob/master/ci-operator/step-registry/stackrox/automation-flavors/ocp-4-e2e/stackrox-automation-flavors-ocp-4-e2e-workflow.yaml
info "Running stackrox OSCI workflow"
info "Worker Nodes: $WORKER_NODE_COUNT"
info "Worker Node Type: $WORKER_NODE_TYPE"
