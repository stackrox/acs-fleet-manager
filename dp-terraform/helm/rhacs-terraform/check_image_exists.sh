#!/bin/bash
set -euo pipefail

org="app-sre"
image="acs-fleet-manager"
tag="$1"
# 40*15s = 10 minutes
retry_attempts=40
sleep_time_sec=15

function image_exists() {
    check_result="$(curl --location --silent --show-error "https://quay.io/api/v1/repository/${org}/${image}/tag?specificTag=${tag}")"
    if [[ ! "$(jq -r '.tags | first | .name' <<<"${check_result}")" == "$tag" ]]; then
        echo >&2 "Image ${image} tag ${tag} does not exist. Received the following response from quay API:"
        echo >&2 "${check_result}"
        return 1
    fi
    return 0
}

if image_exists; then
    exit 0
fi
for attempt in $(seq 1 ${retry_attempts})
do
    echo >&2 "Failed to assert image existence on attempt ${attempt}. Sleeping ${sleep_time_sec}s..."
    sleep ${sleep_time_sec}
    if image_exists; then
        exit 0
    fi
done
echo >&2 "Timed out waiting for the image to appear."
exit 1
