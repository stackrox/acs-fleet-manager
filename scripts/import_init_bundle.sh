#!/usr/bin/env bash
set -eo pipefail

export AWS_AUTH_HELPER="${AWS_AUTH_HELPER:-aws-saml}"
export CHAMBER_SECRET_BACKEND="${CHAMBER_SECRET_BACKEND:-secretsmanager}"

SCRIPTS_DIR=$(realpath "$(dirname "${BASH_SOURCE[0]}")")
# shellcheck source=scripts/lib/external_config.sh
source "$SCRIPTS_DIR/lib/external_config.sh"

usage() {
  echo "Usage: $(basename "$0") [-h | --help] [--dry-run] [--saml-role=<...>] <filename>"
  echo "Imports a given init-bundle file to Secrets Manager."

  if [[ -n "${1:-}" ]]; then
    echo ""
    echo >&2 "Error: $1"
    exit 2
  fi
  exit 0
}

write_secret() {
    local pem
    pem=$(yq -r "$2"  "$init_bundle_file")
    if [[ -n "$dry_run" ]]; then
        echo "(dry-run) Write $1"
        printf "%.15s...\n" "$pem"
    else
        echo "Write $1"
        chamber write secured-cluster --skip-unchanged "$1" -- "$pem"
    fi
}

while (("$#")); do
  case "$1" in
  -h | --help)
    usage
    ;;
  --saml-role=*)
    AWS_SAML_ROLE="${1#*=}"
    export AWS_SAML_ROLE
    shift
    ;;
  --dry-run)
    dry_run=true
    shift
    ;;
  -*)
    usage "Unknown option $1"
    ;;
  *)
    [[ -z "$init_bundle_file" ]] || usage "Exactly one init-bundle file must be specified."
    init_bundle_file="$1"
    shift
    ;;
  esac
done

[[ -n $init_bundle_file ]] || usage "No init-bundle file specified"
[[ -z "$dry_run" ]] || echo "Executing in the dry-run mode."

init_chamber

read -r -p "Will update the secured-cluster secret on behalf of '${AWS_SAML_ROLE}'. Continue? (y/n): " confirmation
if [ "$confirmation" != "y" ]; then
    echo >&2 "Update is cancelled by user."
    exit 1
fi

write_secret CA_CERT '.ca.pem'
write_secret ADMISSION_CONTROL_CERT '.admissionControl.serviceTLS.pem'
write_secret ADMISSION_CONTROL_KEY '.admissionControl.serviceTLS.key'
write_secret COLLECTOR_CERT '.collector.serviceTLS.pem'
write_secret COLLECTOR_KEY '.collector.serviceTLS.key'
write_secret SENSOR_CERT '.sensor.serviceTLS.pem'
write_secret SENSOR_KEY '.sensor.serviceTLS.key'
