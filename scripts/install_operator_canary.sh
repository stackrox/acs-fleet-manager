#!/usr/bin/env bash
set -eo pipefail

export RHACS_TARGETED_OPERATOR_UPGRADES="true"
export RHACS_USE_OPERATORS_CONFIGMAP="true"
export INSTALL_OLM="false"
export INSTALL_OPERATOR="false"
make deploy/bootstrap
make deploy/dev
