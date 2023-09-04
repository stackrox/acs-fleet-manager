#!/usr/bin/env bash
set -eo pipefail

export RHACS_TARGETED_OPERATOR_UPGRADES="true"
export RHACS_STANDALONE_MODE="true"
make deploy/dev
