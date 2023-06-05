#!/usr/bin/env bash

GITROOT="${GITROOT:-"$(git rev-parse --show-toplevel)"}"

# shellcheck source=scripts/lib/log.sh
source "$GITROOT/scripts/lib/log.sh"

export AWS_REGION="${AWS_REGION:-"us-east-1"}"

ensure_tool_installed() {
    make -s -C "$GITROOT" "$GITROOT/bin/$1"
}

add_bin_to_path() {
    if ! [[ ":$PATH:" == *":$GITROOT/bin:"* ]]; then
        export PATH="$GITROOT/bin:$PATH"
    fi
}

init_chamber() {
    add_bin_to_path
    ensure_tool_installed chamber

    AWS_AUTH_HELPER="${AWS_AUTH_HELPER:-none}"
    case $AWS_AUTH_HELPER in
        aws-saml)
            export AWS_PROFILE="saml"
            ensure_tool_installed tools_venv
            # shellcheck source=/dev/null # The script may not exist
            source "$GITROOT/bin/tools_venv/bin/activate"
            # ensure a valid kerberos ticket exist
            if ! klist -s >/dev/null 2>&1; then
                log "Getting a Kerberos ticket"
                kinit
            fi
            aws-saml.py # TODO(ROX-12222): Skip if existing token has not yet expired
        ;;
        none)
            if [[ -z "${AWS_SESSION_TOKEN:-}" ]] || [[ -z "${AWS_ACCESS_KEY_ID:-}" ]] || [[ -z "${AWS_SECRET_ACCESS_KEY:-}" ]]; then
                auth_init_error "Unable to resolve the authentication method"
            fi
        ;;
        *)
            auth_init_error "Unsupported AWS_AUTH_HELPER=$AWS_AUTH_HELPER"
        ;;
    esac
}

auth_init_error() {
    die "Error: $1. Choose one of the following options:
           1) SAML (export AWS_AUTH_HELPER=aws-saml)
           2) Unset AWS_AUTH_HELPER and export AWS_SESSION_TOKEN, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY environment variables"
}

# Loads config from the external storage to the environment and applying a prefix to a variable name (if exists).
load_external_config() {
    local service="$1"
    local prefix="${2:-}"
    local parameter_store_output
    local secrets_manager_output
    parameter_store_output=$(chamber env "$service")
    secrets_manager_output=$(chamber env "$service" -b secretsmanager)
    [[ -z "$parameter_store_output" && -z "$secrets_manager_output" ]] && echo "WARNING: no parameters found under '/$service' of this environment"
    eval "$(printf '%s\n%s' "$parameter_store_output" "$secrets_manager_output" | sed -E "s/(^export +)(.*)/readonly ${prefix}\2/")"
}
