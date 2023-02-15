#!/usr/bin/env bash

GITROOT="$(git rev-parse --show-toplevel)"
export GITROOT
# shellcheck source=/dev/null
source "${GITROOT}/dev/env/scripts/lib.sh"
# shellcheck source=/dev/null
source "${GITROOT}/dev/env/scripts/docker.sh"

init

cat <<EOF
** Preparing ACS MS Test Environment **

Image: ${FLEET_MANAGER_IMAGE}
Namespace: ${ACSMS_NAMESPACE}
Inheriting ImagePullSecrets for Quay.io: ${INHERIT_IMAGEPULLSECRETS}
Installing RHACS Operator: ${INSTALL_OPERATOR}
Operator Source: ${OPERATOR_SOURCE}
Using OLM: ${INSTALL_OLM}
Installing OpenShift Router: ${INSTALL_OPENSHIFT_ROUTER}

EOF

if ! kc_output=$($KUBECTL api-versions 2>&1); then
    die "Error: Sanity check for contacting Kubernetes cluster failed:

Command tried: '$KUBECTL api-versions'
Output:
${kc_output:-(no output)}"
fi

# Create Namespaces.
apply "${MANIFESTS_DIR}/shared"
wait_for_default_service_account "$ACSMS_NAMESPACE"

apply "${MANIFESTS_DIR}/rhacs-operator/00-namespace.yaml"
wait_for_default_service_account "$STACKROX_OPERATOR_NAMESPACE"

# pragma: allowlist nextline secret
if [[ "$INHERIT_IMAGEPULLSECRETS" == "true" ]]; then
    create-imagepullsecrets
    inject_ips "$ACSMS_NAMESPACE" "default" "quay-ips"
    inject_ips "$STACKROX_OPERATOR_NAMESPACE" "default" "quay-ips"
fi

if [[ "$INSTALL_OPENSHIFT_ROUTER" == "true" ]]; then
    log "Installing OpenShift Router"
    apply "${MANIFESTS_DIR}/openshift-router"
fi

if [[ "$INSTALL_OPERATOR" == "true" ]]; then
    install_operator.sh
else
    # We will be running without RHACS operator, but at least install our CRDs.
    apply "${MANIFESTS_DIR}/crds"
fi

if is_local_cluster "$CLUSTER_TYPE"; then
    if [[ ("$INSTALL_OPERATOR" == "true" && "$OPERATOR_SOURCE" == "quay") || "$FLEET_MANAGER_IMAGE" =~ ^quay.io/ ]]; then
        if docker_logged_in "quay.io"; then
            log "Looks like we are already logged into Quay"
        else
            log "Logging into Quay image registry"
            $DOCKER login quay.io -u "$QUAY_USER" --password-stdin <<EOF
$QUAY_TOKEN
EOF
        fi
    fi

    preload_dependency_images
fi

log
log "** Bootstrapping complete **"
log
