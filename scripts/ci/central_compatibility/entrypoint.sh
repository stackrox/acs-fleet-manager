#!/usr/bin/env bash

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "$ROOT_DIR"

source "$ROOT_DIR/scripts/ci/lib.sh"
source "$ROOT_DIR/scripts/lib/log.sh"
source "$ROOT_DIR/dev/env/scripts/lib.sh"

export EMAILSENDER_NS="rhacs"
export CENTRAL_NS="rhacs-tenant"

LOG_DIR="$GITHUB_WORKSPACE/logs"
mkdir "$LOG_DIR"

function log_failure() {
  log "Test failed with status: $EXIT_CODE"
  log "Starting to log cluster resources and container logs into $LOG_DIR directory"

  kubectl describe deploy -n "$EMAILSENDER_NS" > "$LOG_DIR/emailsender-ns-deploy-describe.log"
  kubectl describe pods -n "$EMAILSENDER_NS" -l app=emailsender > "$LOG_DIR/emailsender-ns-describe.log"
  for pod in $(kubectl get pods -n "$EMAILSENDER_NS" -o name); do
    pod_name=$(basename "$pod")
    kubectl logs -n "$EMAILSENDER_NS" "$pod_name" --all-containers --prefix > "$LOG_DIR/${pod_name}.log"
  done

  kubectl describe deploy -n "$CENTRAL_NS" > "$LOG_DIR/central-ns-deploy-describe.log"
  kubectl describe pods -n "$CENTRAL_NS" > "$LOG_DIR/central-ns-pod-describe.log"
  # Log each central namespace pod separately
  for pod in $(kubectl get pods -n "$CENTRAL_NS" -l "app.kubernetes.io/name=stackrox" -o name); do
    pod_name=$(basename "$pod")
    kubectl logs -n "$CENTRAL_NS" "$pod_name" --all-containers --prefix > "$LOG_DIR/${pod_name}.log"
  done
}

bash "$SOURCE_DIR/run_compatibility_test.sh"
EXIT_CODE="$?"
if [ "$EXIT_CODE" -ne "0" ]; then
  log_failure
fi

stat /tmp/pids-port-forward > /dev/null 2>&1 && xargs kill < /tmp/pids-port-forward
exit "$EXIT_CODE"
