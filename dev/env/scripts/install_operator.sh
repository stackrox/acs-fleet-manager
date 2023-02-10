#!/usr/bin/env bash

GITROOT="$(git rev-parse --show-toplevel)"
export GITROOT
# shellcheck source=/dev/null
source "${GITROOT}/dev/env/scripts/lib.sh"
# shellcheck source=/dev/null
source "${GITROOT}/dev/env/scripts/docker.sh"

init

if [[ "$INSTALL_OLM" == "true" ]]; then
    if ! command -v operator-sdk >/dev/null 2>&1; then
        die "Error: Unable to install OLM, operator-sdk executable is not found"
    fi
    # Setup OLM
    if { operator-sdk olm status 2>&1 || true; } | grep -q "no existing installation found"; then
        log "Installing OLM..."
        operator-sdk olm install
    else
        log "OLM already installed..."
    fi
fi

    log "Installing operator"

    apply "${MANIFESTS_DIR}"/rhacs-operator/*.yaml # This installs the operator-group.

    if [[ "$OPERATOR_SOURCE" == "quay" ]]; then
        apply "${MANIFESTS_DIR}"/rhacs-operator/quay/01-catalogsource.yaml
    fi

    # pragma: allowlist nextline secret
    if [[ "$OPERATOR_SOURCE" == "quay" && "$INHERIT_IMAGEPULLSECRETS" == "true" ]]; then
        inject_ips "$STACKROX_OPERATOR_NAMESPACE" "stackrox-operator-test-index" "quay-ips"
    fi

if [[ "$OPERATOR_SOURCE" == "quay" ]]; then
    # Need to wait with the subscription creation until the catalog source has been updated,
    # otherwise the subscription will be in a failed state and not progress.
    # Looks like there is some race which causes the subscription to still fail right after
    # operatorhubio catalog is ready, which is why an additional delay has been added.
    echo "Waiting for CatalogSource to include rhacs-operator..."
    while true; do
        if $KUBECTL -n "$STACKROX_OPERATOR_NAMESPACE" get packagemanifests.packages.operators.coreos.com -o json |
            jq -cer '.items[] | select(.metadata.labels.catalog == "stackrox-operator-test-index" and .metadata.name == "rhacs-operator") | isempty(.) | not' >/dev/null; then
            break
        fi
        sleep 1
    done

    if [[ "$INSTALL_OLM" == "true" ]]; then
        # It seems that before creating the subscription (part of the next apply call) all catalog sources need to be healthy.
        #
        # Installing OLM implies fetching the index from the "operatorhubio" catalog source, which might take some time.
        # If we proceed with creating the subscription for the RHACS Operator immediately and the "operatorhubio" catalog source
        # is not ready get, the subscription can end up in the following state:
        #
        # Conditions:
        #   Message:               all available catalogsources are healthy
        #   Reason:                AllCatalogSourcesHealthy
        #   Status:                False
        #   Type:                  CatalogSourcesUnhealthy
        #   Message:               error using catalog operatorhubio-catalog (in namespace olm): failed to list bundles: rpc error: code = Unavailable desc = connection error: desc = "transport: Error while dialing dial tcp 10.43.96.123:50051: i/o timeout"
        #   Status:                True
        #   Type:                  ResolutionFailed
        #
        # Therefore we wait for the operatorhubio-catalog/registry-server container to become ready.
        wait_for_container_to_become_ready "olm" "olm.catalogSource=operatorhubio-catalog" "registry-server"
    fi

    # This creates the subscription.
    apply "${MANIFESTS_DIR}"/rhacs-operator/quay/*.yaml

    # Apparently we potentially have to wait longer than the default of 60s sometimes...
    wait_for_resource_to_appear "$STACKROX_OPERATOR_NAMESPACE" "serviceaccount" "rhacs-operator-controller-manager" 180
    inject_ips "$STACKROX_OPERATOR_NAMESPACE" "rhacs-operator-controller-manager" "quay-ips"

    # Wait for rhacs-operator pods to be created. Possibly the imagePullSecrets were not picked up yet, which is why we respawn them:
    sleep 2
    $KUBECTL -n "$STACKROX_OPERATOR_NAMESPACE" delete pod -l app=rhacs-operator
elif [[ "$OPERATOR_SOURCE" == "marketplace" ]]; then
    apply "${MANIFESTS_DIR}"/rhacs-operator/marketplace/*.yaml
fi

wait_for_container_to_become_ready "$STACKROX_OPERATOR_NAMESPACE" "app=rhacs-operator" "manager" 900
