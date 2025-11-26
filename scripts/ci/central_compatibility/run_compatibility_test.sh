#!/usr/bin/env bash
set -eux

# Deploy a kind cluster previously to running this script
# This script expects:
# 1. stackrox/stackrox repo to be available at the execution path with directory name stackrox
# 2. acs-fleet-manager repo to be available at the execution path with directory name acs-fleet-manager
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STACKROX_DIR="$(cd "$ROOT_DIR/../stackrox" && pwd)"

EMAILSENDER_NS="rhacs"
CENTRAL_NS="rhacs-tenant"
export ADMIN_PW="letmein"

cd "$ROOT_DIR"

source "$ROOT_DIR/scripts/ci/lib.sh"
source "$ROOT_DIR/scripts/lib/log.sh"
source "$ROOT_DIR/dev/env/scripts/lib.sh"

function pull_to_kind() {
  local img=$1
  local retry="${2:-5}"
  local backoff=30

  for _ in $(seq "$retry"); do
    if docker pull "${img}"; then
      break
    fi

    sleep "$backoff"
  done

  kind load docker-image "${img}"
}

make --no-print-directory -C "$ROOT_DIR" image/build/emailsender
kind load docker-image "$(make --no-print-directory -C "$ROOT_DIR" image-tag/emailsender)"

kubectl create ns $EMAILSENDER_NS -o yaml --dry-run=client | kubectl apply -f -
kubectl create ns $CENTRAL_NS -o yaml --dry-run=client | kubectl apply -f -

make --no-print-directory -C "$ROOT_DIR" deploy/emailsender

log "Emailsender deployed to Kind."

log "Starting to deploy central services..."
# use nightly if GH action running for acs-fleet-manager
# use the stackrox tag otherwise
log "Running for repository: $GITHUB_REPOSITORY"
if [ "$GITHUB_REPOSITORY" = "stackrox/stackrox" ]; then
  ACS_VERSION="$(make --no-print-directory -C "$STACKROX_DIR" tag)"
else
  git -C "$STACKROX_DIR" fetch --tags
  ACS_VERSION="$(git -C "$STACKROX_DIR" tag | grep -E '.*-nightly-[0-9]{8}$' | tail -n 1)"
  git -C "$STACKROX_DIR" checkout "$ACS_VERSION"
fi

log "ACS version: $ACS_VERSION"

IMG_REPO="quay.io/rhacs-eng"
MAIN_IMG_NAME="$IMG_REPO/main"
CENTRAL_DB_IMG_NAME="$IMG_REPO/central-db"

IMAGES_TO_PULL=(
  "$MAIN_IMG_NAME:$ACS_VERSION"
  "$CENTRAL_DB_IMG_NAME:$ACS_VERSION"
)

IMG_WAIT_TIMEOUT_SECONDS="${IMG_WAIT_TIMEOUT_SECONDS:-1200}"
for img in "${IMAGES_TO_PULL[@]}"; do
  wait_for_img "$img" "$IMG_WAIT_TIMEOUT_SECONDS"
  pull_to_kind "$img"
done

TAG="$ACS_VERSION" make --no-print-directory -C "$STACKROX_DIR" cli_host-arch
GOARCH="$(go env GOARCH)"
GOOS="$(go env GOOS)"

ROXCTL="$STACKROX_DIR/bin/${GOOS}_${GOARCH}/roxctl"
# --remove to make this script rerunnable on a local machine
$ROXCTL helm output central-services --remove --output-dir ./central-chart

# Using ACS_VERSION explicitly here since it would otherwise not use the nightly build tag
helm upgrade --install -n $CENTRAL_NS stackrox-central-services ./central-chart \
  -f "${SOURCE_DIR}/central-values.yaml" \
  --set "central.adminPassword.values=$ADMIN_PW" \
  --set "central.image.tag=$ACS_VERSION" \
  --set "central.db.image.tag=$ACS_VERSION" \
  --set "scannerV4.disable=true" \
  --set "scanner.disable=true" # Disabling scanner to reduce resource usage, it is not important for this test

KUBECTL="$(which kubectl)"
wait_for_container_to_become_ready "$CENTRAL_NS" "app=central" "central"
wait_for_container_to_become_ready "$EMAILSENDER_NS" "app=emailsender" "emailsender"

kubectl port-forward -n "$CENTRAL_NS" svc/central 8443:443 >/dev/null &
echo $! >> /tmp/pids-port-forward

cd "$ROOT_DIR"
go test -tags=test_central_compatibility ./emailsender/compatibility
