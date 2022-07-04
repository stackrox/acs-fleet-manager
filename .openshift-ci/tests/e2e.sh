#!/usr/bin/env bash

export GITROOT="$(git rev-parse --show-toplevel)"
source "${GITROOT}/dev/env/scripts/lib.sh"

log
log "** Entrypoint for ACS MS E2E Tests **"
log

log "Retrieving secrets from Vault mount"
shopt -s nullglob
for cred in /var/run/rhacs-ms-e2e-tests/[A-Z]*; do
    secret_name="$(basename "$cred")"
    secret_value="$(cat "$cred")"
    log "Got secret ${secret_name}"
    export "${secret_name}"="${secret_value}"
done

init

log "Image: ${FLEET_MANAGER_IMAGE}"
if [[ "$SPAWN_LOGGER" == "true" ]]; then
    export LOG_DIR=$(mktemp -d)
    log "Log directory: ${LOG_DIR}"
fi

if [[ -n "$OPENSHIFT_CI" ]]; then
    log "Test suite is running in OpenShift CI"
    export GOARGS="-mod=mod" # For some reason we need this in the offical base images.
fi

disable_debugging
case "$AUTH_TYPE" in
OCM)

    log "Refreshing OCM Service Token"
    export OCM_SERVICE_TOKEN=$(ocm token --refresh)
    ;;

STATIC_TOKEN)
    # FLEET_STATIC_TOKEN is the name of the secret in Vault,
    # STATIC_TOKEN is the name expected by the application (when running directly),
    # hence we support both names here.
    STATIC_TOKEN=${STATIC_TOKEN:-}
    export FLEET_STATIC_TOKEN=${FLEET_STATIC_TOKEN:-$STATIC_TOKEN}
    if [[ -z "$FLEET_STATIC_TOKEN" ]]; then
        die "Error: No static token found in the environment.\nPlease set the environment variable STATIC_TOKEN to a valid static token."
    fi
    log "Found static token in the environment"
    ;;
esac

log

if [[ "$INHERIT_IMAGEPULLSECRETS" == "true" ]]; then
    if [[ -z "${QUAY_IO_USERNAME:-}" ]]; then
        die "QUAY_IO_USERNAME needs to be set"
    fi
    if [[ -z "${QUAY_IO_PASSWORD:-}" ]]; then
        die "QUAY_IO_PASSWORD needs to be set"
    fi
fi

enable_debugging_if_necessary
export KUBE_CONFIG=$(assemble-kubeconfig | yq e . -j - | jq -c . -)

# Configuration specific to this e2e test suite:
export ENABLE_DB_PORT_FORWARDING="false"

bootstrap.sh

if [[ -z "$OPENSHIFT_CI" ]]; then
    log "Cleaning up left-over resource (if any)"
    down.sh 2>/dev/null
fi

LOGGER_PID=""
MAIN_LOG="log.txt"

if [[ "$SPAWN_LOGGER" == "true" ]]; then
    log "Spawning logger, log directory is ${LOG_DIR}"
    stern -n "$ACSMS_NAMESPACE" '.*' --color=never --template '[{{.ContainerName}}] {{.Message}}{{"\n"}}' >${LOG_DIR}/${MAIN_LOG} 2>&1 &
    LOGGER_PID=$!
fi

FAIL=0
if ! ${GITROOT}/.openshift-ci/tests/e2e-test.sh; then
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

    log "** BEGIN PODS **"
    $KUBECTL -n "$ACSMS_NAMESPACE" get pods
    $KUBECTL -n "$ACSMS_NAMESPACE" describe pods
    log "** END PODS **"
    log
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

if [[ "$ENABLE_FM_PORT_FORWARDING_DEFAULT" == "true" ]]; then
    port-forwarding stop fleet-manager
fi

if [[ "$ENABLE_DB_PORT_FORWARDING_DEFAULT" == "true" ]]; then
    port-forwarding stop db
fi

exit $FAIL
