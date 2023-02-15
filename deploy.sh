#!/usr/bin/env bash
set -eo pipefail

#colima kubernetes reset
export STACKROX_OPERATOR_VERSION=3.72.0
./dev/env/scripts/install_operator.sh
