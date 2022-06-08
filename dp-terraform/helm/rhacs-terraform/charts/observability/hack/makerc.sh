#!/usr/bin/env bash

set -eu

function wait_for_default_crd() {
    # Since we have to poll and errors are expected, we have to allow errors on this specific command.
    set +e
    local sleep_time=5

    local crd_success=0
    for i in {1..240}; do
        status=$(oc get -n rhacs-observability observabilities.observability.redhat.com observability-stack --ignore-not-found=true -o jsonpath="{.status.stage}{.status.stageStatus}")
        [[ ${status} == "configurationsuccess" ]] && crd_success=1 && break
        echo "polling again in ${sleep_time} seconds"
        sleep ${sleep_time}
    done

    [[ ${crd_success} == 0 ]] && echo 'CRD observability-stack did not reach stage configuration with status success' && exit 1

    set -e

    return 0
}
