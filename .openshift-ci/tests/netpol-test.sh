#!/usr/bin/env bash
set -eo pipefail

GITROOT="$(git rev-parse --show-toplevel)"
export GITROOT
# shellcheck source=/dev/null
source "${GITROOT}/dev/env/scripts/lib.sh"

SERVICE_STARTUP_WAIT_TIME="2m"
CLIENT_STARTUP_WAIT_TIME="1m"

# Tests that a connection is allowed without network policies, and that it's disallowed after applying the policies (in either namespaces)
test_central_connectivity_from_different_namespace() {
    echo "Running ${FUNCNAME[0]}" "$@"

    local CENTRAL_NS="$1"
    local CLIENT_NS="$2"
    local CLIENT_NAME="$3"
    local SERVICE_HOST="$4"

    helm install fake-client "${GITROOT}/test/network-policy/fake-client" --set name="${CLIENT_NAME}" \
        --set service.host="${SERVICE_HOST}" --namespace "${CLIENT_NS}" --create-namespace
    $KUBECTL -n "${CLIENT_NS}" wait --for=condition=Available deployment/"${CLIENT_NAME}" --timeout "${CLIENT_STARTUP_WAIT_TIME}"

    helm install client-netpol "${GITROOT}/fleetshard/pkg/central/charts/data/tenant-resources" --namespace "${CLIENT_NS}" \
        --set secureTenantNetwork=true
    $KUBECTL -n "${CLIENT_NS}" wait --for=condition=Available=false deployment/"${CLIENT_NAME}"

    helm uninstall client-netpol --namespace "${CLIENT_NS}"
    $KUBECTL -n "${CLIENT_NS}" wait --for=condition=Available deployment/"${CLIENT_NAME}"

    helm install central-netpol "${GITROOT}/fleetshard/pkg/central/charts/data/tenant-resources" --namespace "${CENTRAL_NS}" \
        --set secureTenantNetwork=true
    $KUBECTL -n "${CLIENT_NS}" wait --for=condition=Available=false deployment/"${CLIENT_NAME}"

    helm uninstall central-netpol --namespace "${CENTRAL_NS}"
    helm uninstall fake-client --namespace "${CLIENT_NS}"
}

# Tests that a connection is allowed without network policies, and that it's disallowed after applying the policies
test_central_connectivity_from_same_namespace()
{
    echo "Running ${FUNCNAME[0]}" "$@"

    local CLIENT_NS="$1"
    local CLIENT_NAME="$2"
    local SERVICE_HOST="$3"

    helm install fake-client "${GITROOT}/test/network-policy/fake-client" --set name="${CLIENT_NAME}" --namespace "${CLIENT_NS}" \
        --set service.host="${SERVICE_HOST}"
    $KUBECTL -n "${CLIENT_NS}" wait --for=condition=Available deployment/"${CLIENT_NAME}" --timeout "${CLIENT_STARTUP_WAIT_TIME}"

    helm install tenant-netpol "${GITROOT}/fleetshard/pkg/central/charts/data/tenant-resources" --namespace "${CLIENT_NS}" \
        --set secureTenantNetwork=true
    $KUBECTL -n "${CLIENT_NS}" wait --for=condition=Available=false deployment/"${CLIENT_NAME}"

    helm uninstall tenant-netpol --namespace "${CLIENT_NS}"
    helm uninstall fake-client --namespace "${CLIENT_NS}"
}

# Tests that a connection is allowed even with the network policies applied
test_central_connection_allowed()
{
    echo "Running ${FUNCNAME[0]}" "$@"

    local CLIENT_NS="$1"
    local CLIENT_NAME="$2"
    local SERVICE_HOST="$3"

    helm install tenant-netpol "${GITROOT}/fleetshard/pkg/central/charts/data/tenant-resources" --namespace "${CLIENT_NS}" \
        --set secureTenantNetwork=true
    helm install fake-client "${GITROOT}/test/network-policy/fake-client" --set name="${CLIENT_NAME}" --namespace "${CLIENT_NS}" \
        --set service.host="${SERVICE_HOST}"
    $KUBECTL -n "${CLIENT_NS}" wait --for=condition=Available deployment/"${CLIENT_NAME}" --timeout "${CLIENT_STARTUP_WAIT_TIME}"

    helm uninstall tenant-netpol --namespace "${CLIENT_NS}"
    helm uninstall fake-client --namespace "${CLIENT_NS}"
}

# Tests connectivity to a fake Central from various different sources
test_connectivity_to_central() {
    local CENTRAL_NS="rhacs-fake-service"
    local CLIENT_NS="rhacs-fake-client"

    helm install fake-central "${GITROOT}/test/network-policy/fake-service" --namespace "${CENTRAL_NS}" --create-namespace
    $KUBECTL -n "${CENTRAL_NS}" wait --for=condition=Available deployment/central --timeout "${SERVICE_STARTUP_WAIT_TIME}"

    # use the IP to make sure access is denied because of the policy (as opposed to the client not having DNS access)
    local SERVICE_IP
    SERVICE_IP=$($KUBECTL -n rhacs-fake-service get service central-service -o jsonpath='{.spec.clusterIP}')

    # no connections between tenant namespaces should be allowed
    test_central_connectivity_from_different_namespace "${CENTRAL_NS}" "${CLIENT_NS}" central "${SERVICE_IP}"
    test_central_connectivity_from_different_namespace "${CENTRAL_NS}" "${CLIENT_NS}" scanner "${SERVICE_IP}"
    test_central_connectivity_from_different_namespace "${CENTRAL_NS}" "${CLIENT_NS}" scanner-db "${SERVICE_IP}"
    test_central_connectivity_from_different_namespace "${CENTRAL_NS}" "${CLIENT_NS}" scanner-v4-indexer "${SERVICE_IP}"
    test_central_connectivity_from_different_namespace "${CENTRAL_NS}" "${CLIENT_NS}" scanner-v4-matcher "${SERVICE_IP}"
    test_central_connectivity_from_different_namespace "${CENTRAL_NS}" "${CLIENT_NS}" scanner-v4-db "${SERVICE_IP}"
    test_central_connectivity_from_different_namespace "${CENTRAL_NS}" "${CLIENT_NS}" other-app "${SERVICE_IP}"

    # connections from these apps should be allowed within the same namespace
    test_central_connection_allowed "${CENTRAL_NS}" scanner "${SERVICE_IP}"
    test_central_connection_allowed "${CENTRAL_NS}" scanner-v4-indexer "${SERVICE_IP}"
    test_central_connection_allowed "${CENTRAL_NS}" scanner-v4-matcher "${SERVICE_IP}"

    # connections from these apps should *not* be allowed within the same namespace
    test_central_connectivity_from_same_namespace "${CENTRAL_NS}" scanner-db "${SERVICE_IP}"
    test_central_connectivity_from_same_namespace "${CENTRAL_NS}" scanner-v4-db "${SERVICE_IP}"
    test_central_connectivity_from_same_namespace "${CENTRAL_NS}" other-app "${SERVICE_IP}"

    $KUBECTL delete ns "${CENTRAL_NS}"
    $KUBECTL delete ns "${CLIENT_NS}"
}

test_central_connectivity_to_different_namespace() {
    echo "Running ${FUNCNAME[0]}" "$@"

    local SERVICE_NS="$1"
    local CENTRAL_NS="$2"
    local SERVICE_NAME="$3"

    helm install fake-service "${GITROOT}/test/network-policy/fake-service" --namespace "${SERVICE_NS}" --create-namespace \
        --set name="${SERVICE_NAME}"
    $KUBECTL -n "${SERVICE_NS}" wait --for=condition=Available deployment/"${SERVICE_NAME}" --timeout "${SERVICE_STARTUP_WAIT_TIME}"

    # use the IP to make sure access is denied because of the policy (as opposed to the client not having DNS access)
    local SERVICE_IP
    SERVICE_IP=$($KUBECTL -n "${SERVICE_NS}" get service "${SERVICE_NAME}"-service -o jsonpath='{.spec.clusterIP}')

    helm install fake-central "${GITROOT}/test/network-policy/fake-client" --set name=central --namespace "${CENTRAL_NS}" \
        --create-namespace --set service.host="${SERVICE_IP}"
    $KUBECTL -n "${CENTRAL_NS}" wait --for=condition=Available deployment/central --timeout "${CLIENT_STARTUP_WAIT_TIME}"

    helm install central-netpol "${GITROOT}/fleetshard/pkg/central/charts/data/tenant-resources" --namespace "${CENTRAL_NS}" \
        --set secureTenantNetwork=true
    $KUBECTL -n "${CENTRAL_NS}" wait --for=condition=Available=false deployment/central

    helm uninstall central-netpol --namespace "${CENTRAL_NS}"
    helm uninstall fake-central --namespace "${CENTRAL_NS}"
    helm uninstall fake-service --namespace "${SERVICE_NS}"
}

test_central_connectivity_into_same_namespace() {
    test_central_connectivity_to_different_namespace "$1" "$1" "$2"
}

# Tests connectivity from a fake Central to different destinations
test_connectivity_from_central() {
    local SERVICE_NS="rhacs-fake-service"
    local CLIENT_NS="rhacs-fake-client"

    # no connections between tenant namespaces should be allowed
    test_central_connectivity_to_different_namespace "${SERVICE_NS}" "${CLIENT_NS}" central
    test_central_connectivity_to_different_namespace "${SERVICE_NS}" "${CLIENT_NS}" scanner
    test_central_connectivity_to_different_namespace "${SERVICE_NS}" "${CLIENT_NS}" scanner-db
    test_central_connectivity_to_different_namespace "${SERVICE_NS}" "${CLIENT_NS}" scanner-v4-indexer
    test_central_connectivity_to_different_namespace "${SERVICE_NS}" "${CLIENT_NS}" scanner-v4-matcher
    test_central_connectivity_to_different_namespace "${SERVICE_NS}" "${CLIENT_NS}" scanner-v4-db
    test_central_connectivity_to_different_namespace "${SERVICE_NS}" "${CLIENT_NS}" other-app

    # connections to these apps should not be allowed within the same namespace
    test_central_connectivity_into_same_namespace "${SERVICE_NS}" scanner-db
    test_central_connectivity_into_same_namespace "${SERVICE_NS}" scanner-v4-db
    test_central_connectivity_into_same_namespace "${SERVICE_NS}" other-app

    $KUBECTL delete ns "${SERVICE_NS}"
    $KUBECTL delete ns "${CLIENT_NS}"
}

test_connectivity_to_central
test_connectivity_from_central
