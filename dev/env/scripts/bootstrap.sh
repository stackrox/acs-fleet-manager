#!/usr/bin/env bash

GITROOT="$(git rev-parse --show-toplevel)"
export GITROOT
# shellcheck source=/dev/null
source "${GITROOT}/dev/env/scripts/lib.sh"
# shellcheck source=/dev/null
source "${GITROOT}/dev/env/scripts/docker.sh"
# shellcheck source=/dev/null
source "${GITROOT}/scripts/lib/external_config.sh"

init
if [[ "$ENABLE_EXTERNAL_CONFIG" == "true" ]]; then
    init_chamber
    export CHAMBER_SECRET_BACKEND=secretsmanager
else
    add_bin_to_path
    ensure_tool_installed chamber
    export CHAMBER_SECRET_BACKEND=null
fi

log "** Preparing ACSCS Environment **"
print_env

# Retry for up to 30 minutes to contact the Kubernetes cluster
MAX_RETRIES=180  # 30 minutes with 10 second intervals
RETRY_COUNT=0
RETRY_DELAY=10

log "Attempting to contact Kubernetes cluster..."
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if kc_output=$($KUBECTL api-versions 2>&1); then
        log "Successfully contacted Kubernetes cluster"
        break
    fi

    RETRY_COUNT=$((RETRY_COUNT + 1))
    ELAPSED=$((RETRY_COUNT * RETRY_DELAY))
    log "Failed to contact cluster (attempt $RETRY_COUNT/$MAX_RETRIES, elapsed: ${ELAPSED}s). Retrying in ${RETRY_DELAY}s..."
    sleep $RETRY_DELAY
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    die "Error: Sanity check for contacting Kubernetes cluster failed after $((MAX_RETRIES * RETRY_DELAY)) seconds:

Command tried: '$KUBECTL api-versions'
Last output:
${kc_output:-(no output)}"
fi

# Create Namespaces.
apply "${MANIFESTS_DIR}/shared"
create-imagepullsecrets

if [[ "$INSTALL_OPENSHIFT_ROUTER" == "true" ]]; then
    log "Installing OpenShift Router"
    apply "${MANIFESTS_DIR}/openshift-router"
elif [[ "$EXPOSE_OPENSHIFT_ROUTER" == "true" ]]; then
    log "Exposing OpenShift Router"
    oc patch configs.imageregistry.operator.openshift.io/cluster --type merge -p '{"spec":{"defaultRoute":true}}'
else
    log "Skipping installation of OpenShift Router"
fi

if [[ "$INSTALL_VERTICAL_POD_AUTOSCALER" == "true" ]]; then
    log "Installing Vertical Pod Autoscaler"
    apply "${MANIFESTS_DIR}/vertical-pod-autoscaler"
    log "Generating certs for the Vertical Pod Autoscaler Admission Controller"
    "${MANIFESTS_DIR}"/vertical-pod-autoscaler/gencerts.sh
else
    log "Skipping installation of Vertical Pod Autoscaler"
fi

if [[ "$INSTALL_VERTICAL_POD_AUTOSCALER_OLM" == "true" ]]; then
    log "Installing Vertical Pod Autoscaler using OLM"
    apply "${MANIFESTS_DIR}/vertical-pod-autoscaler-olm"
else
    log "Skipping installation of Vertical Pod Autoscaler using OLM"
fi

if [[ "$INSTALL_ARGOCD" == "true" ]]; then
    log "Installing ArgoCD"
    chamber exec gitops -- apply "${MANIFESTS_DIR}/argocd"
    kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
elif [[ "$INSTALL_OPENSHIFT_GITOPS" == "true" ]]; then
    log "Installing Openshift GitOps"
    chamber exec gitops -- apply "${MANIFESTS_DIR}/openshift-gitops"
else
    log "One of ArgoCD or OpenShift GitOps must be installed"
    exit 1
fi

if [[ "$INSTALL_EXTERNAL_SECRETS" == "true" ]]; then # pragma: allowlist secret
    log "Installing External Secrets Operator"

    helm repo add external-secrets https://charts.external-secrets.io 2>/dev/null || true
    helm repo update external-secrets

    helm upgrade --install external-secrets \
        external-secrets/external-secrets \
        --version "$EXTERNAL_SECRETS_VERSION" \
        --namespace rhacs-external-secrets \
        --create-namespace \
        --wait \
        --timeout 5m

    chamber exec external-secrets -- apply "${MANIFESTS_DIR}/external-secrets"
else
    log "Skipping installation of External Secrets Operator"
fi

# skip manifests if openshift cluster using is_openshift_cluster
if ! is_openshift_cluster "$CLUSTER_TYPE"; then
    apply "${MANIFESTS_DIR}/monitoring"
fi

if [[ "$INSTALL_EXTERNAL_DNS" == "true" ]]; then
    log "Installing ExternalDNS for OpenShift"
    apply "${MANIFESTS_DIR}/external-dns-operator"
    wait_for_crd externaldnses.externaldns.olm.openshift.io

    source "${GITROOT}/dev/env/scripts/get-infrastructure-name.sh"
    export EXTERNAL_DNS_NAME=${INFRASTRUCTURE_NAME}
    chamber exec e2e-external-dns -- apply "${MANIFESTS_DIR}/external-dns"
else
    log "Skipping installation of ExternalDNS"
fi

if [[ "$CLUSTER_TYPE"  == "kind" ]]; then
    log "Ensuring operator images exist from dev GitOps config"
    ensure_operator_image_exists.sh
fi

log
log "** Bootstrapping complete **"
log
