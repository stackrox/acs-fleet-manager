#!/usr/bin/env bash
set -eo pipefail

GITROOT="$(git rev-parse --show-toplevel)"
export GITROOT
# shellcheck source=/dev/null
source "${GITROOT}/dev/env/scripts/lib.sh"

# Tests that a connection is allowed without network policies, and that it's disallowed after applying the policies (in either namespaces)
test_central_connectivity_from_different_namespace() {
    local CENTRAL_NS="$1"
    local CLIENT_NS="$2"
    local CLIENT_NAME="$3"

    helm install fake-client "${GITROOT}/test/network-policy/fake-client" --set name="${CLIENT_NAME}" --namespace "${CLIENT_NS}" --create-namespace
    $KUBECTL -n "${CLIENT_NS}" wait --for=condition=Available deployment/"${CLIENT_NAME}"

    helm install client-netpol "${GITROOT}/fleetshard/pkg/central/charts/data/tenant-resources" --namespace "${CLIENT_NS}" --set secureTenantNetwork=true
    $KUBECTL -n "${CLIENT_NS}" wait --for=condition=Available=false deployment/"${CLIENT_NAME}"

    helm uninstall client-netpol --namespace "${CLIENT_NS}"
    $KUBECTL -n "${CLIENT_NS}" wait --for=condition=Available deployment/"${CLIENT_NAME}"

    helm install central-netpol "${GITROOT}/fleetshard/pkg/central/charts/data/tenant-resources" --namespace "${CENTRAL_NS}" --set secureTenantNetwork=true
    $KUBECTL -n "${CLIENT_NS}" wait --for=condition=Available=false deployment/"${CLIENT_NAME}"

    helm uninstall central-netpol --namespace "${CENTRAL_NS}"
    helm uninstall fake-client --namespace "${CLIENT_NS}"
}

# Tests that a connection is allowed without network policies, and that it's disallowed after applying the policies
test_central_connectivity_from_same_namespace()
{
    local CLIENT_NS="$1"
    local CLIENT_NAME="$2"

    helm install fake-client "${GITROOT}/test/network-policy/fake-client" --set name="${CLIENT_NAME}" --namespace "${CLIENT_NS}"
    $KUBECTL -n "${CLIENT_NS}" wait --for=condition=Available deployment/"${CLIENT_NAME}"

    helm install client-netpol "${GITROOT}/fleetshard/pkg/central/charts/data/tenant-resources" --namespace "${CLIENT_NS}" --set secureTenantNetwork=true
    $KUBECTL -n "${CLIENT_NS}" wait --for=condition=Available=false deployment/"${CLIENT_NAME}"

    helm uninstall client-netpol --namespace "${CLIENT_NS}"
    helm uninstall fake-client --namespace "${CLIENT_NS}"
}

test_central_connectivity() {
    local CENTRAL_NS="rhacs-fake-service"
    local CLIENT_NS="rhacs-fake-client"

    helm install fake-central "${GITROOT}/test/network-policy/fake-service" --namespace "${CENTRAL_NS}" --create-namespace
    $KUBECTL -n "${CENTRAL_NS}" wait --for=condition=Available deployment/central

    test_central_connectivity_from_different_namespace "${CENTRAL_NS}" "${CLIENT_NS}" central
    test_central_connectivity_from_different_namespace "${CENTRAL_NS}" "${CLIENT_NS}" scanner
    test_central_connectivity_from_different_namespace "${CENTRAL_NS}" "${CLIENT_NS}" scanner-db
    test_central_connectivity_from_different_namespace "${CENTRAL_NS}" "${CLIENT_NS}" scanner-v4-indexer
    test_central_connectivity_from_different_namespace "${CENTRAL_NS}" "${CLIENT_NS}" scanner-v4-matcher
    test_central_connectivity_from_different_namespace "${CENTRAL_NS}" "${CLIENT_NS}" scanner-v4-db
    test_central_connectivity_from_different_namespace "${CENTRAL_NS}" "${CLIENT_NS}" other-app

    test_central_connectivity_from_same_namespace "${CENTRAL_NS}" scanner-db
    test_central_connectivity_from_same_namespace "${CENTRAL_NS}" scanner-v4-db
    test_central_connectivity_from_same_namespace "${CENTRAL_NS}" other-app

    $KUBECTL delete ns "${CENTRAL_NS}"
    $KUBECTL delete ns "${CLIENT_NS}"
}

test_central_connectivity
