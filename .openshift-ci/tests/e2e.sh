#!/usr/bin/env bash

## This is the main entry point for OpenShift CI. This can also be executed locally against Minikube.
## This script also takes care of OpenShift CI specific initialization
## (e.g. injecting Vault secrets into the environment), spawning loggers, dumping logs at the end.

GITROOT="$(git rev-parse --show-toplevel)"
export GITROOT
# shellcheck source=/dev/null
source "${GITROOT}/dev/env/scripts/lib.sh"

if [[ "${OPENSHIFT_CI:-}" == "true" ]]; then
    # We are running in an OpenShift CI context, configure accordingly.
    log "Executing in OpenShift CI context"
    log "Retrieving secrets from Vault mount"
    shopt -s nullglob
    for cred in /var/run/rhacs-ms-e2e-tests/[A-Z]*; do
        secret_name="$(basename "$cred")"
        secret_value="$(cat "$cred")"
        log "Got secret ${secret_name}"
        export "${secret_name}"="${secret_value}"
    done
    export CLUSTER_TYPE="openshift-ci"
    export STATIC_TOKEN="${FLEET_STATIC_TOKEN:-}"
    export GOARGS="-mod=mod" # For some reason we need this in the offical base images.
fi

init

log
log "** Entrypoint for ACS MS E2E Tests **"
log

log "Cluster type: ${CLUSTER_TYPE}"
log "Image: ${FLEET_MANAGER_IMAGE}"
if [[ "$SPAWN_LOGGER" == "true" ]]; then
    LOG_DIR=$(mktemp -d)
    export LOG_DIR
    log "Log directory: ${LOG_DIR}"
fi

disable_debugging
case "$AUTH_TYPE" in
OCM)

    log "Refreshing OCM Service Token"
    OCM_SERVICE_TOKEN=$(ocm token --refresh)
    export OCM_SERVICE_TOKEN
    ;;

STATIC_TOKEN)
    if [[ -z "$STATIC_TOKEN" ]]; then
        die "Error: No static token found in the environment.\nPlease set the environment variable STATIC_TOKEN to a valid static token."
    fi
    log "Found static token in the environment"
    ;;
esac

log

if [[ "$INHERIT_IMAGEPULLSECRETS" == "true" ]]; then
    if [[ -z "${QUAY_USER:-}" ]]; then
        die "QUAY_USER needs to be set"
    fi
    if [[ -z "${QUAY_TOKEN:-}" ]]; then
        die "QUAY_TOKEN needs to be set"
    fi
fi

enable_debugging_if_necessary

# Configuration specific to this e2e test suite:
export ENABLE_DB_PORT_FORWARDING="false"

bootstrap.sh

if [[ "$CLUSTER_TYPE" != "openshift-ci" ]]; then
    log "Cleaning up left-over resource (if any)"
    down.sh 2>/dev/null
fi

MAIN_LOG="log.txt"

if [[ "$SPAWN_LOGGER" == "true" ]]; then
    log "Spawning logger, log directory is ${LOG_DIR}"
    stern -n "$ACSMS_NAMESPACE" '.*' --color=never --template '[{{.ContainerName}}] {{.Message}}{{"\n"}}' >"${LOG_DIR}/${MAIN_LOG}" 2>&1 &
fi

FAIL=0
if ! "${GITROOT}/.openshift-ci/tests/e2e-test.sh"; then
    FAIL=1
fi

if [[ "$DUMP_LOGS" == "true" ]]; then
    if [[ "$SPAWN_LOGGER" == "true" ]]; then
        log
        log "** BEGIN LOGS **"
        log

        shopt -s nullglob
        for logfile in "${LOG_DIR}"/*; do
            logfile_basename=$(basename "$logfile")
            log
            log "== BEGIN LOG ${logfile_basename} =="
            cat "${logfile}"
            log "== END LOG ${logfile_basename} =="
            log
        done

        log
        log "** END LOGS **"
        log
    fi

    log "** BEGIN ACSMS PODS **"
    $KUBECTL -n "$ACSMS_NAMESPACE" get pods || true
    $KUBECTL -n "$ACSMS_NAMESPACE" describe pods || true
    log "** END ACSMS PODS **"
    log

    log "** BEGIN OPERATOR STATE **"
    $KUBECTL -n "$STACKROX_OPERATOR_NAMESPACE" get pods || true
    $KUBECTL -n "$STACKROX_OPERATOR_NAMESPACE" describe pods || true
    $KUBECTL -n "$STACKROX_OPERATOR_NAMESPACE" get subscriptions || true
    $KUBECTL -n "$STACKROX_OPERATOR_NAMESPACE" describe subscriptions || true
    $KUBECTL -n "$STACKROX_OPERATOR_NAMESPACE" get installplans || true
    $KUBECTL -n "$STACKROX_OPERATOR_NAMESPACE" describe installplans || true
    log "** END OPERATOR STATE **"
    log

    if [[ "$SPAWN_LOGGER" == "true" ]]; then
        log "Logs are in ${LOG_DIR}"
        log
    fi
fi

log "=========="

if [[ $FAIL == 0 ]]; then
    log
    log "** TESTS FINISHED SUCCESSFULLY **"
    log
else
    log
    log "** TESTS FAILED **"
    log
fi

if [[ "$FINAL_TEAR_DOWN" == "true" ]]; then
    down.sh
    delete "${MANIFESTS_DIR}/rhacs-operator" || true
fi

exit $FAIL
