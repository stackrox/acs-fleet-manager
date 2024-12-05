#!/usr/bin/env bash
export CLUSTER_TYPE="infra-openshift"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "$ROOT_DIR"

source "$ROOT_DIR/scripts/ci/lib.sh"
source "$ROOT_DIR/scripts/lib/log.sh"
source "$ROOT_DIR/dev/env/scripts/lib.sh"

bash "$SOURCE_DIR/run_multicluster_tests.sh"
EXIT_CODE="$?"
if [ "$EXIT_CODE" -ne "0" ]; then
  echo "TODO(ROX-27073): add additional logging required here, once tests are actually executed"
fi

stat /tmp/pids-port-forward > /dev/null 2>&1 && xargs kill < /tmp/pids-port-forward
exit "$EXIT_CODE"
