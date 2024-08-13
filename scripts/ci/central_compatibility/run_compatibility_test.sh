#!/usr/bin/env bash

# Deploy a kind cluster previously to running this script
# This script expects:
# 1. stackrox/stackrox repo to be available at the execution path with directory name stackrox
# 2. acs-fleet-manager repo to be available at the execution path with directory name acs-fleet-manager
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EMAILSENDER_HELM_DIR="$ROOT_DIR/dp-terraform/helm/rhacs-terraform"

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

EMAILSENDER_NS="rhacs"
CENTRAL_NS="rhacs-tenant"

kubectl create ns $EMAILSENDER_NS
kubectl create ns $CENTRAL_NS

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
  ACS_VERSION="$( git -C stackrox tag | grep nightly | tail -n 1 )"
  git -C stackrox checkout "$ACS_VERSION"
  SCANNER_VERSION="$(make -C stackrox scanner-tag)"
else
  ACS_VERSION="$(make -C stackrox tag)"
fi


MAIN_IMG="quay.io/rhacs-eng/main:$ACS_VERSION"

IMAGES_TO_PULL=(
  "$MAIN_IMG"
  "quay.io/rhacs-eng/central-db:$ACS_VERSION"
  "quay.io/rhacs-eng/scanner:$SCANNER_VERSION"
  "quay.io/rhacs-eng/scanner-db:$SCANNER_VERSION"
)

for img in "${IMAGES_TO_PULL[@]}"; do
  pull_to_kind "$img"
done

container_id="$(docker create "$MAIN_IMG")"
docker cp "$container_id:/stackrox/roxctl" /tmp/roxctl

export ADMIN_PW="letmein"

/tmp/roxctl helm output central-services --output-dir ./central-chart
helm install -n $CENTRAL_NS stackrox-central-services ./central-chart \
  -f "${SOURCE_DIR}/central-values.yaml" \
  --set "central.adminPassword.values=$ADMIN_PW"

wait_for_container_to_become_ready "$CENTRAL_NS" "application=central" "central"
wait_for_container_to_become_ready "$EMAILSENDER_NS" "application=emailsender" "emailsender"

kubectl port-forward -n $CENTRAL_NS svc/central 8443:443 >/dev/null &

cd acs-fleet-manager
go test -tags=test_central_compatibility ./emailsender/compatibility
