#!/usr/bin/env bash

GITROOT="${GITROOT:-"$(git rev-parse --show-toplevel)"}"
ENABLE_EXTERNAL_CONFIG="${ENABLE_EXTERNAL_CONFIG:-true}"

# shellcheck source=scripts/lib/log.sh
source "$GITROOT/scripts/lib/log.sh"

export AWS_REGION="${AWS_REGION:-"us-east-1"}"

ensure_tool_installed() {
    make -s -C "$GITROOT" "$GITROOT/bin/$1"
}

init_chamber() {
    if ! [[ ":$PATH:" == *":$GITROOT/bin:"* ]]; then
        export PATH="$GITROOT/bin:$PATH"
    fi
    ensure_tool_installed chamber

    if [[ "$ENABLE_EXTERNAL_CONFIG" != "true" ]]; then
        return
    fi

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
        aws-vault)
            export AWS_PROFILE="${AWS_PROFILE:-dev}"
            ensure_tool_installed aws-vault
            ensure_aws_profile_exists
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
           2) aws-vault (export AWS_AUTH_HELPER=aws-vault)
           3) Unset AWS_AUTH_HELPER and export AWS_SESSION_TOKEN, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY environment variables"
}

ensure_aws_profile_exists() {
    if ! aws-vault list --credentials | grep -q "^${AWS_PROFILE}$"; then
        log "Creating profile '$AWS_PROFILE' in AWS Vault"
        if [[ "$AWS_PROFILE" == "dev" ]]; then
            # TODO(ROX-12222): Replace with SSO
            log "Importing dev profile from BitWarden"
            ensure_bitwarden_session_exists
            local aws_creds_json
            aws_creds_json=$(bw get item "23a0e6d6-7b7d-44c8-b8d0-aecc00e1fa0a")
            AWS_ACCESS_KEY_ID=$(jq '.fields[] | select(.name == "AccessKeyID") | .value' --raw-output <<< "$aws_creds_json") \
            AWS_SECRET_ACCESS_KEY=$(jq '.fields[] | select(.name == "SecretAccessKey") | .value' --raw-output <<< "$aws_creds_json") \
                aws-vault add dev --env
        else
            # Input the AWS credentials manually
            aws-vault add "$AWS_PROFILE"
        fi
    fi
}

ensure_bitwarden_session_exists() {
  # Check if we need to get a new BitWarden CLI Session Key.
  if [[ -z "${BW_SESSION:-}" ]]; then
    if bw login --check; then
      # We don't have a session key but we are logged in, so unlock and store the session.
      BW_SESSION=$(bw unlock --raw)
      export BW_SESSION
    else
      # We don't have a session key and are not logged in, so log in and store the session.
      BW_SESSION=$(bw login --raw)
      export BW_SESSION
    fi
  fi
  bw sync -f
}

run_chamber() {
    local args=("$@")
    if [[ "$ENABLE_EXTERNAL_CONFIG" != "true" ]]; then
        # External config disabled. Using 'null' backend for chamber
        args=("-b" "null" "${args[@]}")
    fi
    if [[ "$AWS_AUTH_HELPER" == "aws-vault" ]]; then
        aws-vault exec "${AWS_PROFILE}" -- chamber "${args[@]}"
    else
        chamber "${args[@]}"
    fi
}

# Loads config from the external storage to the environment and applying a prefix to a variable name (if exists).
load_external_config() {
    local service="$1"
    local prefix="${2:-}"
    local parameter_store_output
    local secrets_manager_output
    parameter_store_output=$(run_chamber env "$service")
    [[ -z "$parameter_store_output" ]] && echo "WARNING: no parameters found under '/$service' in AWS Parameter Store of this environment"
    secrets_manager_output=$(run_chamber env "$service" -b secretsmanager)
    [[ -z "$secrets_manager_output" ]] && echo "WARNING: no parameters found under '/$service' in AWS Secrets Manager of this environment"
    eval "$(printf '%s\n%s' "$parameter_store_output" "$secrets_manager_output" | sed -E "s/(^export +)(.*)/readonly ${prefix}\2/")"
}
