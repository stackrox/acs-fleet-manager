#!/usr/bin/env bash

## This is the main entry point for OpenShift CI. This can also be executed locally against Minikube.
## This script also takes care of OpenShift CI specific initialization
## (e.g. injecting Vault secrets into the environment), spawning loggers, dumping logs at the end.

GITROOT="$(git rev-parse --show-toplevel)"
export GITROOT
# shellcheck source=/dev/null
source "${GITROOT}/dev/env/scripts/lib.sh"

export RUN_AUTH_E2E_DEFAULT="false"
export RUN_CENTRAL_E2E_DEFAULT="true"

if [[ "${OPENSHIFT_CI:-}" == "true" ]]; then
    # We are running in an OpenShift CI context, configure accordingly.
    log "Executing in OpenShift CI context"

    log "Retrieving secrets from Vault mount"
    shopt -s nullglob
    for cred in /var/run/rhacs-ms-e2e-tests/[A-Z]*; do
        secret_name="$(basename "$cred")"
        secret_value="$(cat "$cred")"
        case "$secret_name" in
        "IMAGE_PUSH_REGISTRY")
            log "Got secret IMAGE_PUSH_REGISTRY=${secret_value}"
            ;;
        *)
            log "Got secret ${secret_name}"
            ;;
        esac
        export "${secret_name}"="${secret_value}"
    done
    export STATIC_TOKEN="${FLEET_STATIC_TOKEN:-}"
    export STATIC_TOKEN_ADMIN="${FLEET_STATIC_TOKEN_ADMIN:-}"
    export GITHUB_TOKEN="${GITHUB_TOKEN:-}"
    export CLUSTER_TYPE="openshift-ci"
    export GOARGS="-mod=mod" # For some reason we need this in the official base images.
    export GINKGO_FLAGS="--no-color -v"
    # When running in OpenShift CI, ensure we also run the auth E2E tests.
    RUN_AUTH_E2E_DEFAULT="true"
else
    log "Executing in local context"
fi

init

# OpenShift CI provides a per-step ARTIFACT_DIR whose content is uploaded to GCS.
# Locally, use a fresh temp dir per run so reruns never collide or litter the repo.
ARTIFACT_DIR=${ARTIFACT_DIR:-}
if [[ "$SPAWN_LOGGER" == "true" || "$DUMP_LOGS" == "true" ]]; then
    if [[ -z "$ARTIFACT_DIR" ]]; then
        ARTIFACT_DIR=$(mktemp -d "${TMPDIR:-/tmp}/acscs-e2e.XXXXXX")
    fi
    mkdir -p "$ARTIFACT_DIR"
fi
export ARTIFACT_DIR
export RUN_AUTH_E2E=${RUN_AUTH_E2E:-$RUN_AUTH_E2E_DEFAULT}
export RUN_CENTRAL_E2E=${RUN_CENTRAL_E2E:-$RUN_CENTRAL_E2E_DEFAULT}

log
log "** Entrypoint for ACSCS E2E Tests **"
log

log "Cluster type: ${CLUSTER_TYPE}"
log "Cluster name: ${CLUSTER_NAME}"
log "Fleet Manager Image: ${FLEET_MANAGER_IMAGE}"
log "Artifact directory: ${ARTIFACT_DIR:-(none)}"
log "Executing Auth E2E tests: ${RUN_AUTH_E2E}"

if [[ -z "$STATIC_TOKEN" ]]; then
    die "Error: No static token found in the environment.\nPlease set the environment variable STATIC_TOKEN to a valid static token."
fi

if [[ -z "$STATIC_TOKEN_ADMIN" ]]; then
    die "Error: No static admin token found in the environment.\nPlease set the environment variable STATIC_TOKEN_ADMIN to a valid static token."
fi

log

if [[ -z "${QUAY_USER:-}" ]]; then
    die "QUAY_USER needs to be set"
fi
if [[ -z "${QUAY_TOKEN:-}" ]]; then
    die "QUAY_TOKEN needs to be set"
fi

# Configuration specific to this e2e test suite:
export ENABLE_DB_PORT_FORWARDING="false"

if [[ "$SPAWN_LOGGER" == "true" ]]; then
    # Need to create the namespaces prior to spawning the stern loggers.
    # The namespace will be created later anyway, so this pre-creation for stern is best-effort.
    # If it fails, the loggers just won't capture the earliest bootstrap output.
    apply "${MANIFESTS_DIR}/shared/00-namespace.yaml" || true
    sleep 2
    mkdir -p "${ARTIFACT_DIR}/pod-logs"
    log "Spawning logger, log directory is ${ARTIFACT_DIR}/pod-logs"
    stern -n "$ACSCS_NAMESPACE" '.*' --color=never --template '[{{.ContainerName}}] {{.Message}}{{"\n"}}' >"${ARTIFACT_DIR}/pod-logs/namespace-${ACSCS_NAMESPACE}.txt" 2>&1 &
    # Both namespaces default to the same one; a second stern would write to the same file.
    if [[ "$STACKROX_OPERATOR_NAMESPACE" != "$ACSCS_NAMESPACE" ]]; then
        stern -n "$STACKROX_OPERATOR_NAMESPACE" '.*' --color=never --template '[{{.ContainerName}}] {{.Message}}{{"\n"}}' >"${ARTIFACT_DIR}/pod-logs/namespace-${STACKROX_OPERATOR_NAMESPACE}.txt" 2>&1 &
    fi
fi

FAIL=0

if ! "${GITROOT}/.openshift-ci/tests/e2e-test.sh"; then
    FAIL=1
fi

# Terminate loggers.
log "Terminating loggers"
for p in $(jobs -pr); do
    log "Terminating background process ${p}"
    kill "$p" || true
done

log
log "=========="
log

# dump <path relative to ARTIFACT_DIR> <kubectl args...>
# Writes the output of a kubectl call into the artifact directory.
dump() {
    local out="${ARTIFACT_DIR}/$1"
    shift
    mkdir -p "$(dirname "$out")"
    if ! $KUBECTL "$@" >"$out" 2>&1; then
        log "WARN: 'kubectl $*' failed, partial output in ${out}"
    fi
}

# dump_pod_logs <namespace> <path relative to ARTIFACT_DIR>
# Full logs of every pod, plus previous logs of restarted containers.
dump_pod_logs() {
    local ns="$1"
    local dir="$2"
    local pod container
    for pod in $($KUBECTL -n "$ns" get pods -o jsonpath='{.items[*].metadata.name}'); do
        dump "${dir}/${pod}.txt" -n "$ns" logs "$pod" --all-containers --prefix
        for container in $($KUBECTL -n "$ns" get pod "$pod" -o jsonpath='{.status.initContainerStatuses[?(@.restartCount>0)].name} {.status.containerStatuses[?(@.restartCount>0)].name}'); do
            dump "${dir}/${pod}-${container}-previous.txt" -n "$ns" logs "$pod" -c "$container" --previous
        done
    done
}

dump_acscs() {
    dump acscs/describe-pods.txt -n "$ACSCS_NAMESPACE" describe pods
    dump_pod_logs "$ACSCS_NAMESPACE" acscs/logs
}

dump_operator() {
    dump operator/pods.yaml -n "$STACKROX_OPERATOR_NAMESPACE" get pods -o yaml
    dump operator/describe-pods.txt -n "$STACKROX_OPERATOR_NAMESPACE" describe pods
    dump operator/subscriptions.txt -n "$STACKROX_OPERATOR_NAMESPACE" describe subscriptions
    dump operator/installplans.txt -n "$STACKROX_OPERATOR_NAMESPACE" describe installplans
}

dump_external_secrets() {
    local eso_ns="rhacs-external-secrets"
    dump external-secrets/clustersecretstores.yaml get clustersecretstores -o yaml
    dump external-secrets/externalsecrets.yaml -n "$ACSCS_NAMESPACE" get externalsecrets -o yaml
    dump external-secrets/pods.txt -n "$eso_ns" get pods -o wide
    dump_pod_logs "$eso_ns" external-secrets/logs
}

dump_tenant() {
    local ns="$1"
    local dir="tenants/${ns}"
    dump "${dir}/central.yaml" -n "$ns" get centrals.platform.stackrox.io -o yaml
    dump "${dir}/deployments.txt" -n "$ns" describe deployments
    dump "${dir}/describe-pods.txt" -n "$ns" describe pods
    dump "${dir}/events.txt" -n "$ns" get events --sort-by=.lastTimestamp
    dump "${dir}/argocd-app.yaml" -n "$GITOPS_NS" get app "$ns" -o yaml
    dump_pod_logs "$ns" "${dir}/logs"
}

dump_gitops() {
    dump gitops/apps.yaml -n "$GITOPS_NS" get apps -o yaml
}

if [[ "$DUMP_LOGS" == "true" ]]; then
    # Bound every diagnostic call so an unresponsive API server cannot block
    # the script from reaching its final exit status.
    KUBECTL="$KUBECTL --request-timeout=30s"
    GITOPS_NS="openshift-gitops"
    TENANT_NAMESPACES=$($KUBECTL get namespaces -l rhacs.redhat.com/tenant -o jsonpath='{.items[*].metadata.name}' || true)

    # Each job writes to its own files, so they can run concurrently.
    # Concurrency is bounded by the number of sections plus tenant namespaces.
    log "Collecting artifacts in the background into ${ARTIFACT_DIR}"
    dump_acscs &
    dump_operator &
    if [[ "$INSTALL_EXTERNAL_SECRETS" == "true" ]]; then # pragma: allowlist secret
        dump_external_secrets &
    fi
    dump_gitops &
    for ns in $TENANT_NAMESPACES; do
        dump_tenant "$ns" &
    done

    log "** BEGIN CLUSTER SUMMARY **"
    log "--- ${ACSCS_NAMESPACE} pods ---"
    $KUBECTL -n "$ACSCS_NAMESPACE" get pods -o wide || true
    log "--- ${STACKROX_OPERATOR_NAMESPACE} pods ---"
    $KUBECTL -n "$STACKROX_OPERATOR_NAMESPACE" get pods -o wide || true
    $KUBECTL -n "$STACKROX_OPERATOR_NAMESPACE" get subscriptions,installplans || true
    for ns in $TENANT_NAMESPACES; do
        log "--- tenant ${ns} pods ---"
        $KUBECTL -n "$ns" get pods -o wide || true
    done
    log "--- ${GITOPS_NS} ---"
    $KUBECTL -n "$GITOPS_NS" get pods || true
    $KUBECTL -n "$GITOPS_NS" get apps || true
    log "--- warning events ---"
    for ns in "$ACSCS_NAMESPACE" $TENANT_NAMESPACES; do
        $KUBECTL -n "$ns" get events --field-selector type=Warning --sort-by=.lastTimestamp || true
    done
    log "** END CLUSTER SUMMARY **"
    log

    if [[ "$SPAWN_LOGGER" == "true" ]]; then
        shopt -s nullglob
        for logfile in "${ARTIFACT_DIR}/pod-logs"/*; do
            logfile_basename=$(basename "$logfile")
            log "== BEGIN LOG ${logfile_basename} =="
            cat "$logfile"
            log "== END LOG ${logfile_basename} =="
            log
        done
    fi

    log "Waiting for artifact collection to finish"
    wait

    log "** ARTIFACTS **"
    (cd "$ARTIFACT_DIR" && find . -type f -exec ls -lh {} + | awk '{print $5, $9}') || true
    if [[ "${OPENSHIFT_CI:-}" == "true" ]]; then
        log "Artifacts are uploaded to the artifacts directory of this job step."
    else
        log "Artifacts are in ${ARTIFACT_DIR}"
    fi
    log
elif [[ "$SPAWN_LOGGER" == "true" ]]; then
    log "Not dumping logs, logs are in ${ARTIFACT_DIR}/pod-logs"
    log
fi

log
log "=========="
log

if [[ $FAIL == 0 ]]; then
    log
    log "** TESTS FINISHED SUCCESSFULLY **"
    log
else
    log
    log "** TESTS FAILED **"
    log
fi

log "Stopping fleet-manager port-forwarding..."
port-forwarding stop db 5432 || true
port-forwarding stop fleet-manager 8000 || true

log "Killing oc processes forcefully"
pkill -9 '^oc$' || true

exit $FAIL
