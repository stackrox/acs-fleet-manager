#!/usr/bin/env bash
set -eo pipefail

GITROOT="$(git rev-parse --show-toplevel)"
export GITROOT
# shellcheck source=/dev/null
source "${GITROOT}/dev/env/scripts/lib.sh"

CENTRAL_NS="rhacs-fake-service"
SCANNER_NS="rhacs-fake-client"

helm install fake-central "${GITROOT}/test/network-policy/fake-service" --namespace "${CENTRAL_NS}" --create-namespace
$KUBECTL -n "${CENTRAL_NS}" wait --for=condition=Available deployment/central

helm install fake-scanner "${GITROOT}/test/network-policy/fake-client" --namespace "${SCANNER_NS}" --create-namespace
$KUBECTL -n "${SCANNER_NS}" wait --for=condition=Available deployment/scanner

helm install scanner-netpol "${GITROOT}/fleetshard/pkg/central/charts/data/tenant-resources" --namespace "${SCANNER_NS}" --set secureTenantNetwork=true
$KUBECTL -n "${SCANNER_NS}" wait --for=condition=Available=false deployment/scanner

helm uninstall scanner-netpol --namespace "${SCANNER_NS}"
$KUBECTL -n "${SCANNER_NS}" wait --for=condition=Available deployment/scanner

helm install central-netpol "${GITROOT}/fleetshard/pkg/central/charts/data/tenant-resources" --namespace "${CENTRAL_NS}" --set secureTenantNetwork=true
$KUBECTL -n "${SCANNER_NS}" wait --for=condition=Available=false deployment/scanner

$KUBECTL delete ns "${CENTRAL_NS}"
$KUBECTL delete ns "${SCANNER_NS}"
