#!/usr/bin/env bash
set -eo pipefail

export INSTALL_OLM="false"
export INSTALL_OPERATOR="false"
make deploy/bootstrap
make deploy/dev
kubectl set env -n acsms deploy/fleetshard-sync FEATURE_FLAG_UPGRADE_OPERATOR_ENABLED=true
