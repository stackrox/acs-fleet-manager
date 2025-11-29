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
            AWS_SAML_ROLE="${AWS_SAML_ROLE:-"047735621815-poweruser"}"
            export AWS_PROFILE="saml"
            ensure_tool_installed tools_venv
            # shellcheck source=/dev/null # The script may not exist
            source "$GITROOT/bin/tools_venv/bin/activate"
            # ensure a valid kerberos ticket exist
            if ! klist -s >/dev/null 2>&1; then
                log "Getting a Kerberos ticket"
                if ! kinit; then
                    auth_helper_error "kinit failed"
                fi
            fi
            if ! aws-saml.py --target-role "${AWS_SAML_ROLE}"; then
                auth_helper_error "aws-saml.py failed"
            fi
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

auth_helper_error() {
    die "Error: $1. Please refer to the troubleshooting section in docs/development/secret-management.md for a possible cause."
}
