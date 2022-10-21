#!/bin/bash

GITROOT="${GITROOT:-"$(git rev-parse --show-toplevel)"}"
USE_AWS_VAULT=${USE_AWS_VAULT:-true}

export AWS_REGION="${AWS_REGION:-"us-east-1"}"
export AWS_PROFILE=${AWS_PROFILE:-"dev"}

ensure_tool_installed() {
    make -s -C "$GITROOT" "$GITROOT/bin/$1"
}

init_chamber() {
    if ! [[ ":$PATH:" == *":$GITROOT/bin:"* ]]; then
        export PATH="$GITROOT/bin:$PATH"
    fi
    ensure_tool_installed chamber
    if [[ "$USE_AWS_VAULT" = true ]]; then
        ensure_tool_installed aws-vault
        ensure_aws_profile_exists
    elif [[ -z "${AWS_SESSION_TOKEN:-}" ]] || [[ -z "${AWS_ACCESS_KEY_ID:-}" ]] || [[ -z "${AWS_SECRET_ACCESS_KEY:-}" ]]; then
        die "Error: Unable to resolve one of the following environment variables: AWS_SESSION_TOKEN, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY.
            Please set them or use aws-vault by setting USE_AWS_VAULT=true."
    fi
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
    if [[ "$USE_AWS_VAULT" = true ]]; then
        aws-vault exec "${AWS_PROFILE}" -- chamber "$@"
    else
        chamber "$@"
    fi
}
