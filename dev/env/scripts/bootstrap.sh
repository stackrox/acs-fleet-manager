#!/usr/bin/env bash

GITROOT="$(git rev-parse --show-toplevel)"
export GITROOT
# shellcheck source=/dev/null
source "${GITROOT}/dev/env/scripts/lib.sh"
# shellcheck source=/dev/null
source "${GITROOT}/dev/env/scripts/docker.sh"

init

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
    inject_ips "$STACKROX_OPERATOR_NAMESPACE" "default" "quay-ips"
else
    log "Skipping creation of ImagePullSecrets because INHERIT_IMAGEPULLSECRETS is not true"
fi

if [[ "$INSTALL_OPENSHIFT_ROUTER" == "true" ]]; then
    log "Installing OpenShift Router"
    apply "${MANIFESTS_DIR}/openshift-router"
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

# skip manifests if openshift cluster using is_openshift_cluster
if ! is_openshift_cluster "$CLUSTER_TYPE"; then
    apply "${MANIFESTS_DIR}/monitoring"
    apply "${MANIFESTS_DIR}/argocd-operator"
    wait_for_crd "argocds.argoproj.io"
    apply "${MANIFESTS_DIR}/argocd"
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
