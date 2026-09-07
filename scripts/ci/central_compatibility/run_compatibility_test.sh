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

  docker save --platform amd64 "$img" -o "$IMG_TAR_DIR/$imgname.tar"
  kind load image-archive "$IMG_TAR_DIR/$imgname.tar"
}

make --no-print-directory -C "$ROOT_DIR" image/build/emailsender

IMG_TAR_DIR="$(mktemp -d)"

EMAILSENDER_IMAGE="$(make --silent --no-print-directory -C "$ROOT_DIR" image-tag/emailsender)"
docker save --platform amd64 "$EMAILSENDER_IMAGE" -o "$IMG_TAR_DIR/emailsender.tar"
kind load image-archive "$IMG_TAR_DIR/emailsender.tar"

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
IMG_WAIT_TIMEOUT_SECONDS="${IMG_WAIT_TIMEOUT_SECONDS:-1200}"
for imgname in "${IMG_NAMES[@]}"; do
  wait_for_img "$IMG_REPO/$imgname:$ACS_VERSION" "$IMG_WAIT_TIMEOUT_SECONDS"
  pull_to_kind "$IMG_REPO/$imgname:$ACS_VERSION" "$imgname"
done

roxie_envrc="$(mktemp)"

roxie deploy central --verbose --resources auto --tag "${ACS_VERSION}" \
  --envrc "${roxie_envrc}" \
  --set central.namespace="${CENTRAL_NS}" \
  --config "${SOURCE_DIR}/roxie-config.yaml"

# shellcheck source=/dev/null
source "${roxie_envrc}"
export ADMIN_PW="${ROX_ADMIN_PASSWORD}"

KUBECTL="$(which kubectl)"
wait_for_container_to_become_ready "$CENTRAL_NS" "app=central" "central"
wait_for_container_to_become_ready "$EMAILSENDER_NS" "app=emailsender" "emailsender"

kubectl port-forward -n "$CENTRAL_NS" svc/central 8443:443 >/dev/null &
echo $! >> /tmp/pids-port-forward

cd "$ROOT_DIR"
go test -tags=test_central_compatibility ./emailsender/compatibility
