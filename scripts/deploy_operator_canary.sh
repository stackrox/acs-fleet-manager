#!/usr/bin/env bash
set -eo pipefail

export RHACS_TARGETED_OPERATOR_UPGRADES="true"
export RHACS_USE_OPERATORS_CONFIGMAP="true"
make deploy/dev
