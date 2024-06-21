#!/usr/bin/env bash
set -eo pipefail

export AWS_AUTH_HELPER=aws-saml
export CHAMBER_SECRET_BACKEND=secretsmanager

SCRIPTS_DIR=$(realpath "$(dirname "${BASH_SOURCE[0]}")")
# shellcheck source=scripts/lib/external_config.sh
source "$SCRIPTS_DIR/lib/external_config.sh"

usage() {
  echo "Usage: $(basename "$0") <filename>"
  echo "Imports a given init-bundle file to Secrets Manager."

  if [[ -n "${1:-}" ]]; then
    echo ""
    echo >&2 "Error: $1"
    exit 2
  fi
  exit 0
}

init_bundle_file=$1
[[ -n $init_bundle_file ]] || usage "No init-bundle file specified"


init_chamber

write_secret() {
    chamber write secured-cluster --skip-unchanged "$1" -- "$(yq -r "$2"  "$init_bundle_file")"
}

write_secret CA_CERT '.ca.cert'
write_secret ADMISSION_CONTROL_CERT '.admissionControl.serviceTLS.cert'
write_secret ADMISSION_CONTROL_KEY '.admissionControl.serviceTLS.key'
write_secret COLLECTOR_CERT '.collector.serviceTLS.cert'
write_secret COLLECTOR_KEY '.collector.serviceTLS.key'
write_secret SENSOR_CERT '.sensor.serviceTLS.cert'
write_secret SENSOR_KEY '.sensor.serviceTLS.key'
