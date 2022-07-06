#!/usr/bin/env bash

export GITROOT="$(git rev-parse --show-toplevel)"
source "${GITROOT}/dev/env/scripts/lib.sh"
init

up.sh

log "Environment up and running"
log "Waiting for fleet-manager to complete leader election..."
# Don't have a better way yet to wait until fleet-manager has completed the leader election.
$KUBECTL -n "$ACSMS_NAMESPACE" logs -l application=fleet-manager -c fleet-manager -f |
    grep -q --line-buffered --max-count=1 'Running as the leader and starting' || true
sleep 1

log "Next: Executing e2e tests"

FAIL=0
T0=$(date "+%s")
if ! make test/e2e; then
    FAIL=1
fi
T1=$(date "+%s")
DELTA=$(($T1 - $T0))

if [[ $FAIL == 0 ]]; then
    log
    log "** E2E TESTS FINISHED SUCCESSFULLY ($DELTA seconds) **"
    log
else
    log
    log "** E2E TESTS FAILED ($DELTA seconds) **"
    log
fi

if [[ "$FINAL_TEAR_DOWN" == "true" ]]; then
    down.sh
fi

exit $FAIL
