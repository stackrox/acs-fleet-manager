#!/usr/bin/env bash

set -eou pipefail

function log() {
    echo "${1:-}" >&2
}

function log_exit() {
    log "${1:-}"

    exit 1
}

function usage() {
    log "
Usage:
    migrate-authn-z.sh MANDATORY [OPTION]
MANDATORY:
    --central-list-path     The path to the with list of central instances that should be migrated.
OPTION:
    --help                  Prints help information.
Example:
    migrate-authn-z.sh --central-list gabi-list.json
"
}

function usage_exit() {
    usage

    exit 1
}

function check_command() {
    local cmd="${1:-}"

    echo "- Looking for '${cmd}'"
    command -v "${cmd}" || log_exit "-- Command '${cmd}' required."
    echo "- Found '${cmd}'!"
}

function check_dependencies() {
    check_command jq
    check_command realpath
}

function migrate_all_centrals() {
    echo "-- Processing all ACS instance from the list"

    local central_list_path="${1:-}"
    [[ "${central_list_path}" = "" ]] && log "Error: Parameter 'central_list_path' is empty." && usage_exit

    local script_dir
    script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

    local binary_file_path
    binary_file_path=$(realpath "${script_dir}/../../dataplane-migrators")

    # 1. read CSV file and process central one by one. So that we can log results nicely.
    # Used query:
    # ./stage-action.sh gabi "SELECT id,name,CONCAT('https://acs-',id,'.',host),issuer FROM central_requests WHERE deleted_at IS NULL ORDER BY created_at DESC;" | jq '.result[1:][] | @csv ' --raw-output > stage.csv
    while read -r csv_line; do
        # id,name,CONCAT('https://acs-',id,'.',host)
        IFS="," read -r csv_id csv_name csv_host csv_issuer <<<"${csv_line}"

        # trim quotes
        csv_id=$(echo "${csv_id}" | xargs)
        csv_name=$(echo "${csv_name}" | xargs)
        csv_host=$(echo "${csv_host}" | xargs)
        csv_issuer=$(echo "${csv_issuer}" | xargs)
        echo ">>> Migrate central ID: ${csv_id} - with name: ${csv_name}"

        echo ">>> Calling command:"
        echo ">>> " "${binary_file_path}" migrate authn-z \
            --id "${csv_id}" \
            --name "${csv_name}" \
            --url "${csv_host}" \
            --issuer "${csv_issuer}"

        # execute migration
        ${binary_file_path} migrate authn-z \
            --id "${csv_id}" \
            --name "${csv_name}" \
            --url "${csv_host}" \
            --issuer "${csv_issuer}"

        echo ">>> Done"
    done <"${central_list_path}"

    echo "-- Done"
}

function main() {
    local central_list_path=""

    while [[ -n "${1:-}" ]]; do
        case "${1}" in
        "--central-list-path")
            central_list_path="${2:-}"
            shift
            ;;
        "--help")
            usage_exit
            ;;
        *)
            log "Error: Unknown parameter: ${1:-}"
            usage_exit
            ;;
        esac

        if ! shift; then
            log "Error: Missing parameter argument."
            usage_exit
        fi
    done

    check_dependencies

    [[ "${central_list_path}" = "" ]] && log "Error: Command option '--central-list-path' is mandatory." && usage_exit

    central_list_path=$(realpath "${central_list_path}")
    [[ ! -f "${central_list_path}" ]] && log "Error: File provided in option '--central-list-path' does not exist." && usage_exit

    migrate_all_centrals "${central_list_path}" "{stage}"
}

main "$@"
