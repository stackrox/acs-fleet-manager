#!/bin/bash

function invoke_helm() {
    local -r dir="${1}"
    shift
    local -r release="${1}"
    shift

    helm repo add external-secrets "https://charts.external-secrets.io/" --force-update

    # Build the external dependencies like the external-secrets helm chart bundle.
    helm dependencies build "${dir}"

    if [[ "${ENVIRONMENT}" == "dev" ]]; then
        # Dev env is special, as there is no real dev cluster. Instead
        # we just run lint to smoke test the chart.
        helm lint "${dir}" "$@"
    else
        if [[ "${HELM_DRY_RUN:-}" == "true" ]]; then
            HELM_FLAGS="--dry-run"
        else
            # Install CRDs if they did not exist in the previous revisions or update them
            # This is necessary because Helm treats CRDs differently.
            # Links:
            #   - https://helm.sh/docs/chart_best_practices/custom_resource_definitions
            #   - https://github.com/helm/community/blob/main/hips/hip-0011.md
            #   - https://github.com/helm/helm/issues/11969
            if [ -d "${dir}/crds" ]; then
                kubectl apply -f "${dir}/crds"
            fi
        fi
        if [[ "${HELM_DEBUG:-}" == "true" ]]; then
            HELM_FLAGS="${HELM_FLAGS:-} --debug"
        fi
        # shellcheck disable=SC2086
        helm upgrade "${release}" "${dir}" ${HELM_FLAGS:-} \
          --install --create-namespace "$@"
  fi
}
