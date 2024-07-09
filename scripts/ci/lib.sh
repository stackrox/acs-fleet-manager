#!/usr/bin/env bash

# A library of CI related reusable bash functions
SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

# shellcheck source=scripts/lib/log.sh
source "$SCRIPTS_ROOT/scripts/lib/log.sh"

ci_export() {
    if [[ "$#" -ne 2 ]]; then
        die "missing args. usage: ci_export <env-name> <env-value>"
    fi

    local env_name="$1"
    local env_value="$2"

    if command -v cci-export >/dev/null; then
        cci-export "$env_name" "$env_value"
    else
        export "$env_name"="$env_value"
    fi
}

# set_ci_shared_export() - for openshift-ci this is state shared between steps.
set_ci_shared_export() {
    if [[ "$#" -ne 2 ]]; then
        die "missing args. usage: set_ci_shared_export <env-name> <env-value>"
    fi

    ci_export "$@"

    local env_name="$1"
    local env_value="$2"

    echo "export ${env_name}=${env_value}" | tee -a "${SHARED_DIR:-/tmp}/shared_env"
}
