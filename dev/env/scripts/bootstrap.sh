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
wait_for_default_service_account "$ACSCS_NAMESPACE"

# pragma: allowlist nextline secret
if [[ "$INHERIT_IMAGEPULLSECRETS" == "true" ]]; then
    create-imagepullsecrets
    inject_ips "$ACSCS_NAMESPACE" "default" "quay-ips"
else
    log "Skipping creation of ImagePullSecrets because INHERIT_IMAGEPULLSECRETS is not true"
fi

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

# skip manifests if openshift cluster using is_openshift_cluster
if ! is_openshift_cluster "$CLUSTER_TYPE"; then
    apply "${MANIFESTS_DIR}/monitoring"
fi

apply "${MANIFESTS_DIR}/addons"

if is_local_cluster "$CLUSTER_TYPE"; then
    if [[  "$FLEET_MANAGER_IMAGE" =~ ^quay.io/ ]]; then
        if docker_logged_in "quay.io"; then
            log "Looks like we are already logged into Quay"
        else
            log "Logging into Quay image registry"
            $DOCKER login quay.io -u "$QUAY_USER" --password-stdin <<EOF
$QUAY_TOKEN
EOF
        fi
    fi

    if [[ "$CLUSTER_TYPE"  == "kind" ]]; then
        log "Ensuring operator images exist from dev GitOps config"
        ensure_operator_image_exists.sh
    fi
fi

log
log "** Bootstrapping complete **"
log
