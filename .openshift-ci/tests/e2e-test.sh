#!/usr/bin/env bash

export GITROOT="$(git rev-parse --show-toplevel)"
export PATH="$GITROOT/dev/env/scripts:${PATH}"
source "${GITROOT}/dev/env/scripts/lib.sh"
init

up.sh

log "Environment up and running"
log "Waiting for fleet-manager to complete leader election..."
# Don't have a better way yet to wait until fleet-manager has completed the leader election.
$KUBECTL -n "$ACSMS_NAMESPACE" logs -l io.kompose.service=fleet-manager -c fleet-manager -f |
    grep --line-buffered 'Running as the leader and starting' |
    head -1 >/dev/null || true
sleep 1

log "Next: Executing e2e tests"

export RUN_E2E=true

FAIL=0
if ! go test -bench -v -count=1 ./e2e/...; then
    FAIL=1
fi

if [[ $FAIL == 0 ]]; then
    log
    log "** TESTS FINISHED SUCCESSFULLY **"
    log
else
    log
    log "** TESTS FAILED **"
    log
fi

exit $FAIL
