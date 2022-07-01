#!/usr/bin/env bash

export GITROOT="$(git rev-parse --show-toplevel)"
export PATH="$GITROOT/dev/env/scripts:${PATH}"
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

if [[ -n "$OPENSHIFT_CI" ]]; then
    log "Test suite is running in OpenShift CI"
else
    export ENABLE_DB_PORT_FORWARDING="true"
    export ENABLE_FM_PORT_FORWARDING="true"
fi

if [[ -z "$OPENSHIFT_CI" ]]; then
    # Will be replaced with static auth.
    log "Refreshing OCM token (currently injected into fleetshard-sync)"
    disable_debugging
    OCM_TOKEN=$(ocm token)
    export OCM_TOKEN
else
    # This will cause fleetshard-sync to fail.
    # Will be replaced with static auth.
    export OCM_TOKEN="not-a-token"
fi

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

LOG_DIR=$(mktemp -d)
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

if [[ "$SPAWN_LOGGER" == "true" ]]; then
    log "Terminating logger"
    kill "$LOGGER_PID" || true
    sleep 1

    log "** BEGIN LOG **"
    cat "${LOG_DIR}/${MAIN_LOG}"
    log "** END LOG **"
    log
fi

log "** BEGIN PODS **"
$KUBECTL -n "$ACSMS_NAMESPACE" get pods
$KUBECTL -n "$ACSMS_NAMESPACE" describe pods
log "** END PODS **"
log

log "** BEGIN FLEET-MANAGER POD LOGS **"
$KUBECTL -n "$ACSMS_NAMESPACE" logs io.kompose.service=fleet-manager
log "** END FLEET-MANAGER POD LOGS **"
log

log "** BEGIN FLEETSHARD-SYNC POD LOGS **"
$KUBECTL -n "$ACSMS_NAMESPACE" logs io.kompose.service=fleetshard-sync
log "** END FLEETSHARD-SYNC POD LOGS **"
log

exit $FAIL
