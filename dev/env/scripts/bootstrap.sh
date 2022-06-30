#!/usr/bin/env bash

export GITROOT="$(git rev-parse --show-toplevel)"
source "${GITROOT}/dev/env/scripts/lib.sh"
init

if [[ "$CLUSTER_TYPE" == "minikube" ]]; then
    if ! minikube status >/dev/null; then
        unset DEBUG
        minikube start --memory=5G \
            --cpus=4 \
            --apiserver-port=8443 \
            --embed-certs=true \
            --apiserver-names=minikube \
            --delete-on-failure=true
    else
        if ! kc_output=$($KUBECTL get nodes 2>&1); then
            die "Sanity check for contacting Kubernetes cluster failed: ${kc_output}"
        fi
    fi
fi

# Create Namespaces.
apply "${MANIFESTS_DIR}/shared"
echo "Waiting for default service account to be created in namespace '${ACSMS_NAMESPACE}'"
for i in $(seq 10); do
    if $KUBECTL -n "$ACSMS_NAMESPACE" get sa default 2>/dev/null >&2; then
        break
    fi
    sleep 1
done

apply "${MANIFESTS_DIR}/rhacs-operator/00-namespace.yaml"
echo "Waiting for default service account to be created in namespace '${STACKROX_OPERATOR_NAMESPACE}'"
for i in $(seq 10); do
    if $KUBECTL -n "$STACKROX_OPERATOR_NAMESPACE" get sa default 2>/dev/null >&2; then
        break
    fi
    sleep 1
done

sleep 1 # I have seen failures without an extra delay between the above check and the patching of ServiceAccounts.

# TODO: use a function.
if [[ "$INHERIT_IMAGEPULLSECRETS" == "true" ]]; then
    create-imagepullsecrets-interactive
    log "Patching ServiceAccount ${ACSMS_NAMESPACE}/default to use Quay.io imagePullSecrets"
    $KUBECTL -n "$ACSMS_NAMESPACE" patch sa default -p '"imagePullSecrets": [{"name": "quay-ips" }]'
fi

if [[ "$INHERIT_IMAGEPULLSECRETS" == "true" ]]; then
    echo "Patching ServiceAccount ${STACKROX_OPERATOR_NAMESPACE}/default to use imagePullSecrets"
    $KUBECTL -n "$STACKROX_OPERATOR_NAMESPACE" patch sa default -p '"imagePullSecrets": [{"name": "quay-ips" }]'
fi

if [[ "$INSTALL_OPERATOR" == "true" ]]; then
    if [[ "$INSTALL_OLM" == "true" ]]; then
        # Setup OLM
        operator-sdk olm install
    fi

    if [[ "$OPERATOR_SOURCE" == "quay" ]]; then
        apply "${MANIFESTS_DIR}"/rhacs-operator/quay/01*
    fi

    if [[ "$OPERATOR_SOURCE" == "quay" && "$INHERIT_IMAGEPULLSECRETS" == "true" ]]; then
        echo "Patching ServiceAccount ${STACKROX_OPERATOR_NAMESPACE}/stackrox-operator-test-index to use imagePullSecrets"
        $KUBECTL -n "$STACKROX_OPERATOR_NAMESPACE" patch sa stackrox-operator-test-index -p '"imagePullSecrets": [{"name": "quay-ips" }]'
    fi

    if [[ "$OPERATOR_SOURCE" == "quay" ]]; then
        # Need to wait with the subscription creation until the catalog source has been updated,
        # otherwise the subscription will be in a failed state and not progress.
        # Looks like there is some race which causes the subscription to still fail right after
        # operatorhubio catalog is ready, which is why an additional delay has been added.
        $KUBECTL -n olm wait --timeout=120s --for=condition=ready pod -l olm.catalogSource=operatorhubio-catalog
        echo "Waiting for CatalogSource to include rhacs-operator..."
        while true; do
            $KUBECTL -n "$STACKROX_OPERATOR_NAMESPACE" get packagemanifests.packages.operators.coreos.com -o json |
                jq -r '.items[].metadata.name' | grep -q '^rhacs-operator$' && break
            sleep 1
        done

        echo "Waiting for CatalogSource to include bundles from operatorhubio-catalog..."
        while true; do
            $KUBECTL -n "$STACKROX_OPERATOR_NAMESPACE" get packagemanifests.packages.operators.coreos.com -o json |
                jq -r '.items[].metadata.labels.catalog' | grep -q '^operatorhubio-catalog$' && break
            sleep 1
        done
    fi

    if [[ "$OPERATOR_SOURCE" == "quay" ]]; then
        apply "${MANIFESTS_DIR}"/rhacs-operator/quay/0[23]*
    elif [[ "$OPERATOR_SOURCE" == "marketplace" ]]; then
        apply "${MANIFESTS_DIR}"/rhacs-operator/marketplace/0[23]*
    fi

    if [[ "$OPERATOR_SOURCE" == "quay" ]]; then
        echo "Waiting for SA to appear..."
        while true; do
            $KUBECTL -n "$STACKROX_OPERATOR_NAMESPACE" get serviceaccount rhacs-operator-controller-manager >/dev/null 2>&1 && break
            sleep 1
        done

        echo "Patching ServiceAccount rhacs-operator-controller-manager to use imagePullSecrets"
        if [[ "$INHERIT_IMAGEPULLSECRETS" == "true" ]]; then
            $KUBECTL -n "$STACKROX_OPERATOR_NAMESPACE" patch sa rhacs-operator-controller-manager -p '"imagePullSecrets": [{"name": "quay-ips" }]'
        fi

        sleep 2 # Wait for rhacs-operator pods to be created. Possibly the imagePullSecrets were not picked up yet, which is why we respawn them:
        $KUBECTL -n "$STACKROX_OPERATOR_NAMESPACE" delete pod -l app=rhacs-operator
    fi

    sleep 1
    $KUBECTL -n "$STACKROX_OPERATOR_NAMESPACE" wait --timeout=120s --for=condition=ready pod -l app=rhacs-operator
else
    # We will be running without RHACS operator, but at least install our CRDs.
    apply "${MANIFESTS_DIR}/crds"
fi

load_image_into_minikube() {
    local img="$1"

    if $MINIKUBE image ls | grep -q "^${img}$"; then
        true
    else
        $DOCKER pull "${img}" && $DOCKER save "${img}" | $MINIKUBE ssh --native-ssh=false docker load
    fi
}

if [[ "$CLUSTER_TYPE" == "minikube" ]]; then
    log "Preloading images into minikube..."
    # Preload images required by Central installation.
    load_image_into_minikube "${IMAGE_REGISTRY}/scanner:${SCANNER_VERSION}"
    load_image_into_minikube "${IMAGE_REGISTRY}/scanner-db:${SCANNER_VERSION}"
    load_image_into_minikube "${IMAGE_REGISTRY}/main:${CENTRAL_VERSION}"
    log "Images preloaded"
fi

log "** Bootstrapping complete **"
