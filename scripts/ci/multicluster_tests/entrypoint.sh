#!/usr/bin/env bash
export CLUSTER_TYPE="infra-openshift"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "$ROOT_DIR"

source "$ROOT_DIR/scripts/ci/lib.sh"
source "$ROOT_DIR/scripts/lib/log.sh"
source "$ROOT_DIR/dev/env/scripts/lib.sh"

acscs_namespace="rhacs"

log_common_cluster_resources() {
  local cluster_name=$1

  log "***** START logging resources for: $cluster_name *****"
  kubectl describe deploy -n "$acscs_namespace"
  kubectl describe pods -n "$acscs_namespace"
  kubectl get routes -n "$acscs_namespace"
  kubectl logs -n "$acscs_namespace" --prefix --all-containers -l app=fleetshard-sync
  log "***** END logging resources for: $cluster_name *****"
}

log_failure() {
  local step_name=$1
  log "$step_name Failed with status: $EXIT_CODE"
  log "Starting to log cluster resources and container logs"

  export KUBECONFIG=$CLUSTER_1_KUBECONFIG
  log_common_cluster_resources "CLUSTER_1"
  kubectl logs -n "$acscs_namespace" --prefix --all-containers -l app=fleet-manager

  export KUBECONFIG=$CLUSTER_2_KUBECONFIG
  log_common_cluster_resources "CLUSTER_2"
}


# shellcheck source=scripts/lib/external_config.sh
source "${ROOT_DIR}/scripts/lib/external_config.sh"

# Executing this with all necessary credentials stored in AWS secretsmanager on ACSCS dev account
# Name: "github-multicluster-test"
# created and updated only manually by ACSCS engineerse
init_chamber
secrets=$(chamber env "github-multicluster-test" --backend secretsmanager)
eval "$secrets"

KUBECTL="$(which kubectl)"
export KUBECTL
log "kubectl version:"
kubectl version --client

bash "$SOURCE_DIR/deploy.sh"
EXIT_CODE="$?"
if [ "$EXIT_CODE" -ne "0" ]; then
  log_failure Deploy
  stat /tmp/pids-port-forward > /dev/null 2>&1 && xargs kill < /tmp/pids-port-forward
  exit 1
fi

FM_URL="https://$(KUBECONFIG=$CLUSTER_1_KUBECONFIG kubectl get routes -n rhacs fleet-manager -o yaml | yq .spec.host)"
export FM_URL

make test/e2e/multicluster
EXIT_CODE="$?"
if [ "$EXIT_CODE" -ne "0" ]; then
  log_failure Test
fi

stat /tmp/pids-port-forward > /dev/null 2>&1 && xargs kill < /tmp/pids-port-forward
exit "$EXIT_CODE"
