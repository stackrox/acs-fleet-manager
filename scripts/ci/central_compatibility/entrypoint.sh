#!/usr/bin/env bash

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "$ROOT_DIR"

source "$ROOT_DIR/scripts/ci/lib.sh"
source "$ROOT_DIR/scripts/lib/log.sh"
source "$ROOT_DIR/dev/env/scripts/lib.sh"

export EMAILSENDER_NS="rhacs"
export CENTRAL_NS="rhacs-tenant"
mkdir logs
function log_failure() {
  log "Test failed with status: $EXIT_CODE"
  log "Starting to log cluster resources and container logs..."

  log "***** START EMAILSENDER RESOURCES *****"
  kubectl describe deploy -n "$EMAILSENDER_NS" emailsender > logs/emailsender-deploy-describe.log
  kubectl describe pods -n "$EMAILSENDER_NS" -l app=emailsender > logs/emailsender-pod-describe.log
  kubectl logs -n "$EMAILSENDER_NS" --prefix --all-containers -l app=emailsender > logs/emailsender-all-pods.log

  kubectl describe deploy -n "$EMAILSENDER_NS" emailsender-db > logs/emailsender-db-deploy-describe.log
  kubectl describe pods -n "$EMAILSENDER_NS" -l app=emailsender-db > logs/emailsender-db-pod-describe.log
  kubectl logs -n "$EMAILSENDER_NS" --prefix --all-containers -l app=emailsender-db > logs/emailsender-db-all-pods.log
  log "***** END EMAILSENDER RESOURCES *****"

  log "***** START STACKROX KUBERNETES RESOURCES *****"
  kubectl describe deploy -n "$CENTRAL_NS" > logs/central-ns-deploy-describe.log
  kubectl describe pods -n "$CENTRAL_NS" > logs/central-ns-pod-describe.log
  kubectl logs -n "$CENTRAL_NS" --prefix --all-containers -l "app.kubernetes.io/name=stackrox" > logs/central-ns-all-pods.log
  log "***** END STACKROX KUBERNETES RESOURCES *****"
}

bash "$SOURCE_DIR/run_compatibility_test.sh"
EXIT_CODE="$?"
if [ "$EXIT_CODE" -ne "0" ]; then
  log_failure
fi

stat /tmp/pids-port-forward > /dev/null 2>&1 && xargs kill < /tmp/pids-port-forward
exit "$EXIT_CODE"
