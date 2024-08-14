#!/usr/bin/env bash

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd $ROOT_DIR

source "$ROOT_DIR/scripts/ci/lib.sh"
source "$ROOT_DIR/scripts/lib/log.sh"
source "$ROOT_DIR/dev/env/scripts/lib.sh"

export EMAILSENDER_NS="rhacs"
export CENTRAL_NS="rhacs-tenant"

function log_failure() {
  log "Test failed with status: $?" 
  log "Starting to log cluster resources and container logs..."

  log "***** START EMAILSENDER RESOURCES *****"
    kubectl describe deploy -n "$EMAILSENDER_NS" emailsender
    kubectl describe pods -n "$EMAILSENDER_NS" -l app=emailsender 
    kubectl logs -n "$EMAILSENDER_NS" --prefix --all-containers -l app=emailsender

    kubectl describe deploy -n "$EMAILSENDER_NS" emailsender-db
    kubectl describe pods -n "$EMAILSENDER_NS" -l app=emailsender-db
    kubectl logs -n "$EMAILSENDER_NS" --prefix --all-containers -l app=emailsender-db
  log "***** END EMAILSENDER RESOURCES *****"

  log "***** START STACKROX KUBERNETES RESOURCES *****"
    kubectl describe deploy -n "$CENTRAL_NS"
    kubectl describe pods -n "$CENTRAL_NS"
    kubectl logs -n "$CENTRAL_NS" --prefix --all-containers -l "app.kubernetes.io/name=stackrox"
  log "***** END STACKROX KUBERNETES RESOURCES *****"
}

touch pids-port-forward

bash "$SOURCE_DIR/run_compatibilty_test.sh"
EXIT_CODE="$?"

if [ "$EXIT_CODE" -ne 0 ]; then
  log_failure
fi

cat pids-port-forwad | xargs kill
exit $EXIT_CODE


