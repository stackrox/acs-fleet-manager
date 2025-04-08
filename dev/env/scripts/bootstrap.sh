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

if ! kc_output=$($KUBECTL api-versions 2>&1); then
    die "Error: Sanity check for contacting Kubernetes cluster failed:

Command tried: '$KUBECTL api-versions'
Output:
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
    wait_for_crd "applications.argoproj.io"
    wait_for_resource_to_appear "$ARGOCD_NAMESPACE" "appprojects.argoproj.io" "default"
    log "Installing External Secrets Operator"
    apply "${MANIFESTS_DIR}/external-secrets/application"
    wait_for_crd "clustersecretstores.external-secrets.io"
    wait_for_container_to_appear "rhacs-external-secrets" "app.kubernetes.io/name=external-secrets-webhook" "webhook"
    chamber exec external-secrets -- apply "${MANIFESTS_DIR}/external-secrets"
else
    log "Skipping installation of External Secrets Operator"
fi

# skip manifests if openshift cluster using is_openshift_cluster
if ! is_openshift_cluster "$CLUSTER_TYPE"; then
    apply "${MANIFESTS_DIR}/monitoring"
fi

apply "${MANIFESTS_DIR}/addons"

if [[ "$CLUSTER_TYPE"  == "kind" ]]; then
    log "Ensuring operator images exist from dev GitOps config"
    ensure_operator_image_exists.sh
fi

log
log "** Bootstrapping complete **"
log
