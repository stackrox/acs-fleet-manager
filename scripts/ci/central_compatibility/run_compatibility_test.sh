#!/usr/bin/env bash
set -eu

# Deploy a kind cluster previously to running this script
# This script expects:
# 1. stackrox/stackrox repo to be available at the execution path with directory name stackrox
# 2. acs-fleet-manager repo to be available at the execution path with directory name acs-fleet-manager
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

EMAILSENDER_NS="rhacs"
CENTRAL_NS="rhacs-tenant"
export ADMIN_PW="letmein"

cd "$ROOT_DIR"

source "$ROOT_DIR/scripts/ci/lib.sh"
source "$ROOT_DIR/scripts/lib/log.sh"
source "$ROOT_DIR/dev/env/scripts/lib.sh"

function pull_to_kind() {
  local img=$1
  local imgname=$2
  local retry="${3:-5}"
  local backoff=30

  for _ in $(seq "$retry"); do
    if docker pull "${img}"; then
      break
    fi

    sleep "$backoff"
  done

  docker save --platform amd64 $img -o "$imgname.tar"
  kind load image-archive "$imgname.tar"
}

make --no-print-directory -C "$ROOT_DIR" image/build/emailsender

EMAILSENDER_IMAGE="$(make --silent --no-print-directory -C "$ROOT_DIR" image-tag/emailsender)"
docker save --platform amd64 "$EMAILSENDER_IMAGE" -o emailsender.tar
kind load image-archive emailsender.tar

kubectl create ns $EMAILSENDER_NS -o yaml --dry-run=client | kubectl apply -f -
kubectl create ns $CENTRAL_NS -o yaml --dry-run=client | kubectl apply -f -

make --no-print-directory -C "$ROOT_DIR" deploy/emailsender

log "Emailsender deployed to Kind."

log "Starting to deploy central services..."
# use nightly if GH action running for acs-fleet-manager
# use the stackrox tag otherwise
GITHUB_REPOSITORY=${GITHUB_REPOSITORY:-stackrox/acs-fleet-manager}
log "Running for repository: $GITHUB_REPOSITORY"
if [ "$GITHUB_REPOSITORY" = "stackrox/stackrox" ]; then
  STACKROX_DIR="$(cd "$ROOT_DIR/../stackrox" && pwd)"
  ACS_VERSION="$(make --silent --no-print-directory -C "$STACKROX_DIR" tag)"
else
  ACS_VERSION="$(git ls-remote --tags https://github.com/stackrox/stackrox | grep -E '.*-nightly-[0-9]{8}$' | awk '{print $2}' | sed 's|refs/tags/||' | sort -V | tail -n 1)"
fi

log "ACS version: $ACS_VERSION"

IMG_REPO="quay.io/rhacs-eng"
IMG_NAMES=(
  "main"
  "central-db"
)
MAIN_IMG="$IMG_REPO:main:$ACS_VERSION"
IMG_WAIT_TIMEOUT_SECONDS="${IMG_WAIT_TIMEOUT_SECONDS:-1200}"
for imgname in "${IMAGES_NAMES[@]}"; do
  wait_for_img "$IMG_REPO/$imgname:$ACS_VERSION" "$IMG_WAIT_TIMEOUT_SECONDS"
  pull_to_kind "$IMG_REPO/$imgname:$ACS_VERSION" "$imgname"
done

ROXCTL="docker run --rm --user $(id -u):$(id -g) -v $(pwd):/tmp/stackrox-charts/ $MAIN_IMG"
# --remove to make this script rerunnable on a local machine
$ROXCTL helm output central-services --image-defaults opensource --remove --output-dir /tmp/stackrox-charts/central-chart

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
