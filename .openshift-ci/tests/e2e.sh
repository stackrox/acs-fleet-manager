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
    export CLUSTER_TYPE="openshift-ci"
    export GOARGS="-mod=mod" # For some reason we need this in the official base images.
    export GINKGO_FLAGS="--no-color -v"
    # When running in OpenShift CI, ensure we also run the auth E2E tests.
    RUN_AUTH_E2E_DEFAULT="true"
    export INHERIT_IMAGEPULLSECRETS="true" # pragma: allowlist secret
else
    log "Executing in local context"
fi

init

LOG_DIR=${LOG_DIR:-}
if [[ "$SPAWN_LOGGER" == "true" && "$LOG_DIR" == "" ]]; then
    LOG_DIR=$(mktemp -d)
fi
export LOG_DIR
export RUN_AUTH_E2E=${RUN_AUTH_E2E:-$RUN_AUTH_E2E_DEFAULT}
export RUN_CENTRAL_E2E=${RUN_CENTRAL_E2E:-$RUN_CENTRAL_E2E_DEFAULT}

log
log "** Entrypoint for ACSCS E2E Tests **"
log

log "Cluster type: ${CLUSTER_TYPE}"
log "Cluster name: ${CLUSTER_NAME}"
log "Fleet Manager Image: ${FLEET_MANAGER_IMAGE}"
log "Fleetshard Operator Image: ${FLEETSHARD_OPERATOR_IMAGE:-latest}"
log "Log directory: ${LOG_DIR:-(none)}"
log "Executing Auth E2E tests: ${RUN_AUTH_E2E}"

# If auth E2E tests shall be run, ensure we have all authentication related secrets correctly set up.
if [[ "$RUN_AUTH_E2E" == "true" ]]; then
    log "Setting up authentication related environment variables for auth E2E tests"

    OCM_OFFLINE_TOKEN=${OCM_OFFLINE_TOKEN:-}
    if [[ -z "$OCM_OFFLINE_TOKEN" ]]; then
        die "Error: OCM_OFFLINE_TOKEN not set, which is required for execution of Auth E2E tests"
    fi

    # Ensure we set the OCM refresh token
    ocm login --token "${OCM_OFFLINE_TOKEN}"
    OCM_TOKEN=$(ocm token --refresh)
    export OCM_TOKEN

    # The RH SSO secrets are correctly set up within vault, the tests will be skipped if they are empty.
else
    log "Skipping setup of authentication related environment variables for auth E2E tests because RUN_AUTH_E2E is not set to true"
fi

if [[ -z "$STATIC_TOKEN" ]]; then
    die "Error: No static token found in the environment.\nPlease set the environment variable STATIC_TOKEN to a valid static token."
fi

if [[ -z "$STATIC_TOKEN_ADMIN" ]]; then
    die "Error: No static admin token found in the environment.\nPlease set the environment variable STATIC_TOKEN_ADMIN to a valid static token."
fi

log

if [[ "$INHERIT_IMAGEPULLSECRETS" == "true" ]]; then # pragma: allowlist secret
    log "INHERIT_IMAGEPULLSECRETS is true, verifying that QUAY_USER and QUAY_TOKEN are set"
    if [[ -z "${QUAY_USER:-}" ]]; then
        die "QUAY_USER needs to be set"
    fi
    if [[ -z "${QUAY_TOKEN:-}" ]]; then
        die "QUAY_TOKEN needs to be set"
    fi
fi

# Configuration specific to this e2e test suite:
export ENABLE_DB_PORT_FORWARDING="false"

if [[ "$SPAWN_LOGGER" == "true" ]]; then
    # Need to create the namespaces prior to spawning the stern loggers.
    apply "${MANIFESTS_DIR}/shared/00-namespace.yaml"
    sleep 2
    log "Spawning logger, log directory is ${LOG_DIR}"
    stern -n "$ACSCS_NAMESPACE" '.*' --color=never --template '[{{.ContainerName}}] {{.Message}}{{"\n"}}' >"${LOG_DIR}/namespace-${ACSCS_NAMESPACE}.txt" 2>&1 &
    stern -n "$STACKROX_OPERATOR_NAMESPACE" '.*' --color=never --template '[{{.ContainerName}}] {{.Message}}{{"\n"}}' >"${LOG_DIR}/namespace-${STACKROX_OPERATOR_NAMESPACE}.txt" 2>&1 &
fi

FAIL=0
if ! "${GITROOT}/.openshift-ci/tests/netpol-test.sh"; then
    FAIL=1
fi

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

if [[ "$DUMP_LOGS" == "true" ]]; then
    log "Dumping logs"
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

    log "** BEGIN ACSCS PODS **"
    $KUBECTL -n "$ACSCS_NAMESPACE" get pods || true
    $KUBECTL -n "$ACSCS_NAMESPACE" describe pods || true
    log "** END ACSCS PODS **"
    log

    log "** BEGIN OPERATOR STATE **"
    $KUBECTL -n "$STACKROX_OPERATOR_NAMESPACE" get pods || true
    $KUBECTL -n "$STACKROX_OPERATOR_NAMESPACE" get pods -o yaml || true
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
else
    if [[ "$SPAWN_LOGGER" == "true" ]]; then
        log "Not dumping logs, logs are in ${LOG_DIR}"
        log
    fi
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
