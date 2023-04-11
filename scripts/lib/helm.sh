#!/bin/bash

function invoke_helm() {
    local -r dir="${1}"
    shift
    local -r release="${1}"
    shift

    if [[ "${ENVIRONMENT}" == "dev" ]]; then
        # Dev env is special, as there is no real dev cluster. Instead
        # we just run lint to smoke test the chart.
        helm lint "${dir}" "$@"
    else
        if [[ "${HELM_DRY_RUN:-}" == "true" ]]; then
            HELM_FLAGS="--dry-run"
        fi
        if [[ "${HELM_DEBUG:-}" == "true" ]]; then
            HELM_FLAGS="${HELM_FLAGS:-} --debug"
        fi
        # shellcheck disable=SC2086
        helm upgrade "${release}" "${dir}" ${HELM_FLAGS:-} \
          --install --create-namespace "$@"
  fi
}
