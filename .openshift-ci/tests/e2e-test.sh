#!/usr/bin/env bash

GITROOT="$(git rev-parse --show-toplevel)"
export GITROOT
# shellcheck source=/dev/null
source "${GITROOT}/dev/env/scripts/lib.sh"

bootstrap.sh

log "Setting up e2e test environment"

if [[ "$CLUSTER_TYPE" != "openshift-ci" ]]; then
    log "Cleaning up left-over resource (if any)"
    down.sh 2>/dev/null
else
    log "Skipping cleanup of left-over resources because CLUSTER_TYPE is openshift-ci"
fi

up.sh

log "Environment up and running"

FAIL=0
if [[ "$SKIP_TESTS" == "true" ]]; then
    log "Skipping tests"
else
    log "Next: Executing e2e tests"

    T0=$(date "+%s")
    if ! make test/e2e; then
        FAIL=1
    fi
    T1=$(date "+%s")
    DELTA=$((T1 - T0))

    if [[ $FAIL == 0 ]]; then
        log
        log "** E2E TESTS FINISHED SUCCESSFULLY ($DELTA seconds) **"
        log
    else
        log
        log "** E2E TESTS FAILED ($DELTA seconds) **"
        log
    fi
fi

log "** Exiting with status code $FAIL **"
exit $FAIL
