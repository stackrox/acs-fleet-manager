#!/usr/bin/env bash
set -euo pipefail

# Deploy a kind cluster previously to running this script
# This script expects:
# 1. stackrox/stackrox repo to be available at the execution path with directory name stackrox
# 2. acs-fleet-manager repo to be available at the execution path with directory name acs-fleet-manager
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EMAILSENDER_HELM_DIR="$ROOT_DIR/dp-terraform/helm/rhacs-terraform"
STACKROX_DIR="$(pwd)/stackrox"
EMAILSENDER_NS="rhacs"
CENTRAL_NS="rhacs-tenant"
export ADMIN_PW="letmein"

cd "$ROOT_DIR"

source "$ROOT_DIR/scripts/ci/lib.sh"
source "$ROOT_DIR/scripts/lib/log.sh"
source "$ROOT_DIR/dev/env/scripts/lib.sh"

function pull_to_kind() {
  local img=$1
  docker pull "${img}"
  kind load docker-image "${img}"
}

EMAILSENDER_IMG_TAG="$(make -C "$ROOT_DIR" tag)"
EMAILSENDER_IMG_NAME="$(make -C "$ROOT_DIR" image-name/emailsender)"
EMAILSENDER_IMG="$(make -C "$ROOT_DIR" image-tag/emailsender)"
make -C "$ROOT_DIR" image/build/emailsender
kind load docker-image "${EMAILSENDER_IMG}"

kubectl create ns $EMAILSENDER_NS -o yaml --dry-run=client | kubectl apply -f -
kubectl create ns $CENTRAL_NS -o yaml --dry-run=client | kubectl apply -f -

helm dependency build "${EMAILSENDER_HELM_DIR}"
# Render emailsender kubernetes resources
helm template --namespace "${EMAILSENDER_NS}" \
  -f "${SOURCE_DIR}/emailsender-values.yaml" "${EMAILSENDER_HELM_DIR}" \
  --set emailsender.image.repo="${EMAILSENDER_IMG_NAME}" \
  --set emailsender.image.tag="${EMAILSENDER_IMG_TAG}" \
  | yq e '. | select(.metadata.name == "emailsender")' \
  > emailsender-manifests.yaml

kubectl apply -f emailsender-manifests.yaml
kubectl apply -f "${SOURCE_DIR}/emailsender-db.yaml"

# use nightly if GH action running for acs-fleet-manager
# use the stackrox tag otherwise
if [ "$GITHUB_REPOSITORY" = "stackrox/acs-fleet-manager" ]; then
  ACS_VERSION="$( git -C "$STACKROX_DIR" tag | grep nightly | tail -n 1 )"
  git -C "$STACKROX_DIR" checkout "$ACS_VERSION"
  SCANNER_VERSION="$(make -C "$STACKROX_DIR" scanner-tag)"
else
  ACS_VERSION="$(make -C "$STACKROX_DIR" tag)"
fi

IMG_REPO="quay.io/rhacs-eng"
MAIN_IMG_NAME="$IMG_REPO/main"
CENTRAL_DB_IMG_NAME="$IMG_REPO/central-db"
SCANNER_IMG_NAME="$IMG_REPO/scanner"
SCANNER_DB_IMG_NAME="$IMG_REPO/scanner-db"

IMAGES_TO_PULL=(
  "$MAIN_IMG_NAME:$ACS_VERSION"
  "$CENTRAL_DB_IMG_NAME:$ACS_VERSION"
  "$SCANNER_IMG_NAME:$SCANNER_VERSION"
  "$SCANNER_DB_IMG_NAME:$SCANNER_VERSION"
)

for img in "${IMAGES_TO_PULL[@]}"; do
  pull_to_kind "$img"
done

make -C "$STACKROX_DIR" cli_host-arch cli-install

# --remove to make this script rerunnable on a local machine
roxctl helm output central-services --remove --output-dir ./central-chart

# Using ACS_VERSION explicitly here since it would otherwise not use the nightly build tag
helm upgrade --install -n $CENTRAL_NS stackrox-central-services ./central-chart \
  -f "${SOURCE_DIR}/central-values.yaml" \
  --set "central.adminPassword.values=$ADMIN_PW" \
  --set "central.image.tag=$ACS_VERSION" \
  --set "central.db.image.tag=$ACS_VERSION"

KUBECTL="$(which kubectl)"
wait_for_container_to_become_ready "$CENTRAL_NS" "app=central" "central"
wait_for_container_to_become_ready "$EMAILSENDER_NS" "app=emailsender" "emailsender"

kubectl port-forward -n "$CENTRAL_NS" svc/central 8443:443 >/dev/null &
echo $! >> /tmp/pids-port-forward

cd "$ROOT_DIR"
go test -tags=test_central_compatibility ./emailsender/compatibility
