#!/usr/bin/env bash
set -eo pipefail

kind load docker-image quay.io/rhacs-eng/stackrox-operator:4.2.0
kind load docker-image quay.io/rhacs-eng/stackrox-operator:4.1.0

export RHACS_TARGETED_OPERATOR_UPGRADES="true"
export INSTALL_OLM="false"
export INSTALL_OPERATOR="false"
make deploy/bootstrap
make deploy/dev
