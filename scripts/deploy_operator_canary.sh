#!/usr/bin/env bash
set -eo pipefail

export RHACS_TARGETED_OPERATOR_UPGRADES="true"
make deploy/dev
