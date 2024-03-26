# shellcheck shell=bash

GITROOT_DEFAULT=$(git rev-parse --show-toplevel)
export GITROOT=${GITROOT:-$GITROOT_DEFAULT}

# shellcheck source=/dev/null
source "$GITROOT/scripts/lib/log.sh"

# export scripts if not in path
if ! command -v bootstrap.sh >/dev/null 2>&1; then
    export PATH="$GITROOT/dev/env/scripts:${PATH}"
fi

try_kubectl() {
    local kubectl
    if command -v kubectl >/dev/null 2>&1; then
        kubectl="kubectl"
    elif command -v oc >/dev/null 2>&1; then
        kubectl="oc"
    else
        log "Error: Neither 'kubectl' nor 'oc' found." >&2
        return 1
    fi

    if $kubectl "$@"; then
        return 0
    else
        return 1
    fi
}

get_current_cluster_name() {
    local cluster_name
    cluster_name=$(try_kubectl config view --minify=true | yq e '.clusters[].name' -)
    if [[ -z "$cluster_name" ]]; then
        log "Error: Failed to retrieve cluster name, please set CLUSTER_NAME" >&2
        return 1
    fi
    echo "$cluster_name"
}

init() {
    set -eu -o pipefail

    # For reading the defaults we need access to the
    if [[ -z "${CLUSTER_NAME:-}" ]]; then
        CLUSTER_NAME=$(get_current_cluster_name)
        if [[ -z "$CLUSTER_NAME" ]]; then
            die "Error: Failed to retrieve cluster name."
        fi
    fi
    export CLUSTER_NAME

    for env_file in "${GITROOT}/dev/env/defaults/"*.env; do
        # shellcheck source=/dev/null
        source "$env_file"
    done

    if ! command -v bootstrap.sh >/dev/null 2>&1; then
        export PATH="$GITROOT/dev/env/scripts:${PATH}"
    fi

    available_cluster_types=$(find "${GITROOT}/dev/env/defaults" -maxdepth 1 -type d -name "cluster-type-*" -print0 | xargs -0 -n1 basename | sed -e 's/^cluster-type-//;' | sort | paste -sd "," -)

    export CLUSTER_TYPE="${CLUSTER_TYPE:-$CLUSTER_TYPE_DEFAULT}"
    if [[ -z "$CLUSTER_TYPE" ]]; then
        die "Error: CLUSTER_TYPE not set and could not be figured out. Please make sure that it is initialized properly. Available cluster types: ${available_cluster_types}"
    elif [[ ! "$available_cluster_types" =~ (^|,)"$CLUSTER_TYPE"($|,) ]]; then
        die "Error: CLUSTER_TYPE '${CLUSTER_TYPE}' is not supported. Available cluster types: ${available_cluster_types}"
    fi

    for env_file in "${GITROOT}/dev/env/defaults/cluster-type-${CLUSTER_TYPE}/"*; do
        # shellcheck source=/dev/null
        source "$env_file"
    done

    export ENABLE_EXTERNAL_CONFIG="${ENABLE_EXTERNAL_CONFIG:-$ENABLE_EXTERNAL_CONFIG_DEFAULT}"
    export AWS_AUTH_HELPER="${AWS_AUTH_HELPER:-$AWS_AUTH_HELPER_DEFAULT}"

    export KUBECTL=${KUBECTL:-$KUBECTL_DEFAULT}
    export ACSCS_NAMESPACE="${ACSCS_NAMESPACE:-$ACSCS_NAMESPACE_DEFAULT}"
    export DOCKER=${DOCKER:-$DOCKER_DEFAULT}
    export KIND=${KIND:-$KIND_DEFAULT}
    export IMAGE_REGISTRY="${IMAGE_REGISTRY:-$IMAGE_REGISTRY_DEFAULT}"
    IMAGE_REGISTRY_HOST=$(if [[ "$IMAGE_REGISTRY" =~ ^[^/]*\.[^/]*/ ]]; then echo "$IMAGE_REGISTRY" | cut -d / -f 1; fi)
    export IMAGE_REGISTRY_HOST
    export CENTRAL_VERSION="${CENTRAL_VERSION:-$CENTRAL_VERSION_DEFAULT}"
    export SCANNER_VERSION="${SCANNER_VERSION:-$SCANNER_VERSION_DEFAULT}"
    export STACKROX_OPERATOR_NAMESPACE="${STACKROX_OPERATOR_NAMESPACE:-$STACKROX_OPERATOR_NAMESPACE_DEFAULT}"
    export INSTALL_OPENSHIFT_ROUTER="${INSTALL_OPENSHIFT_ROUTER:-$INSTALL_OPENSHIFT_ROUTER_DEFAULT}"
    export OCM_SERVICE_CLIENT_ID=${OCM_SERVICE_CLIENT_ID:-$OCM_SERVICE_CLIENT_ID_DEFAULT}
    export OCM_SERVICE_CLIENT_SECRET=${OCM_SERVICE_CLIENT_SECRET:-$OCM_SERVICE_CLIENT_SECRET_DEFAULT}
    export OCM_SERVICE_TOKEN=${OCM_SERVICE_TOKEN:-$OCM_SERVICE_TOKEN_DEFAULT}
    export OCM_ADDON_SERVICE_CLIENT_ID=${OCM_ADDON_SERVICE_CLIENT_ID:-$OCM_ADDON_SERVICE_CLIENT_ID_DEFAULT}
    export OCM_ADDON_SERVICE_CLIENT_SECRET=${OCM_ADDON_SERVICE_CLIENT_SECRET:-$OCM_ADDON_SERVICE_CLIENT_SECRET_DEFAULT}
    export OCM_ADDON_SERVICE_TOKEN=${OCM_ADDON_SERVICE_TOKEN:-$OCM_ADDON_SERVICE_TOKEN_DEFAULT}
    export SENTRY_KEY=${SENTRY_KEY:-$SENTRY_KEY_DEFAULT}
    export AWS_ACCESS_KEY=${AWS_ACCESS_KEY:-$AWS_ACCESS_KEY_DEFAULT}
    export AWS_ACCOUNT_ID=${AWS_ACCOUNT_ID:-$AWS_ACCOUNT_ID_DEFAULT}
    export AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY:-$AWS_SECRET_ACCESS_KEY_DEFAULT}
    export SSO_CLIENT_ID=${SSO_CLIENT_ID:-$SSO_CLIENT_ID_DEFAULT}
    export SSO_CLIENT_SECRET=${SSO_CLIENT_SECRET:-$SSO_CLIENT_SECRET_DEFAULT}
    export OSD_IDP_SSO_CLIENT_ID=${OSD_IDP_SSO_CLIENT_ID:-$OSD_IDP_SSO_CLIENT_ID_DEFAULT}
    export OSD_IDP_SSO_CLIENT_SECRET=${OSD_IDP_SSO_CLIENT_SECRET:-$OSD_IDP_SSO_CLIENT_SECRET_DEFAULT}
    export ROUTE53_ACCESS_KEY=${ROUTE53_ACCESS_KEY:-$ROUTE53_ACCESS_KEY_DEFAULT}
    export ROUTE53_SECRET_ACCESS_KEY=${ROUTE53_SECRET_ACCESS_KEY:-$ROUTE53_SECRET_ACCESS_KEY_DEFAULT}
    export OBSERVABILITY_CONFIG_ACCESS_TOKEN=${OBSERVABILITY_CONFIG_ACCESS_TOKEN:-$OBSERVABILITY_CONFIG_ACCESS_TOKEN_DEFAULT}
    export INHERIT_IMAGEPULLSECRETS=${INHERIT_IMAGEPULLSECRETS:-$INHERIT_IMAGEPULLSECRETS_DEFAULT}
    export SPAWN_LOGGER=${SPAWN_LOGGER:-$SPAWN_LOGGER_DEFAULT}
    export DUMP_LOGS=${DUMP_LOGS:-$DUMP_LOGS_DEFAULT}
    export OPERATOR_SOURCE=${OPERATOR_SOURCE:-$OPERATOR_SOURCE_DEFAULT}
    export INSTALL_OLM=${INSTALL_OLM:-$INSTALL_OLM_DEFAULT}
    export ENABLE_DB_PORT_FORWARDING=${ENABLE_DB_PORT_FORWARDING:-$ENABLE_DB_PORT_FORWARDING_DEFAULT}
    export ENABLE_FM_PORT_FORWARDING=${ENABLE_FM_PORT_FORWARDING:-$ENABLE_FM_PORT_FORWARDING_DEFAULT}
    export FLEETSHARD_SYNC_RESOURCES=${FLEETSHARD_SYNC_RESOURCES:-$FLEETSHARD_SYNC_RESOURCES_DEFAULT}
    export DOCKER_CONFIG=${DOCKER_CONFIG:-$DOCKER_CONFIG_DEFAULT}
    [[ -d "$DOCKER_CONFIG" ]] || mkdir -p "$DOCKER_CONFIG"
    export SKIP_TESTS=${SKIP_TESTS:-$SKIP_TESTS_DEFAULT}
    export ENABLE_CENTRAL_EXTERNAL_CERTIFICATE=${ENABLE_CENTRAL_EXTERNAL_CERTIFICATE:-$ENABLE_CENTRAL_EXTERNAL_CERTIFICATE_DEFAULT}
    export FLEET_MANAGER_IMAGE=${FLEET_MANAGER_IMAGE:-$FLEET_MANAGER_IMAGE_DEFAULT}

    FLEETSHARD_SYNC_CONTAINER_COMMAND_DEFAULT="/usr/local/bin/fleetshard-sync"
    export FLEETSHARD_SYNC_CONTAINER_COMMAND=${FLEETSHARD_SYNC_CONTAINER_COMMAND:-$FLEETSHARD_SYNC_CONTAINER_COMMAND_DEFAULT}

    if [[ "$FLEET_MANAGER_IMAGE" == "" ]]; then
        FLEET_MANAGER_IMAGE=$(make -s -C "$GITROOT" full-image-tag)
        log "FLEET_MANAGER_IMAGE not set, using ${FLEET_MANAGER_IMAGE}"
    fi

    if [[ "$ENABLE_CENTRAL_EXTERNAL_CERTIFICATE" != "false" && ("$ROUTE53_ACCESS_KEY" == "" || "$ROUTE53_SECRET_ACCESS_KEY" == "") ]]; then
        log "setting ENABLE_CENTRAL_EXTERNAL_CERTIFICATE to false since no Route53 credentials were provided"
        ENABLE_CENTRAL_EXTERNAL_CERTIFICATE=false
    fi

    if [[ "$CLUSTER_TYPE" == "minikube" ]]; then
        eval "$(minikube docker-env)"
    fi

}

print_env() {
    cat <<EOF
** Environment **
CLUSTER_TYPE: ${CLUSTER_TYPE}
CLUSTER_NAME: ${CLUSTER_NAME}
ENABLE_EXTERNAL_CONFIG: ${ENABLE_EXTERNAL_CONFIG}
AWS_AUTH_HELPER: ${AWS_AUTH_HELPER}
KUBECTL: ${KUBECTL}
ACSCS_NAMESPACE: ${ACSCS_NAMESPACE}
DOCKER: ${DOCKER}
KIND: ${KIND}
IMAGE_REGISTRY: ${IMAGE_REGISTRY}
IMAGE_REGISTRY_HOST: ${IMAGE_REGISTRY_HOST}
CENTRAL_VERSION: ${CENTRAL_VERSION}
SCANNER_VERSION: ${SCANNER_VERSION}
STACKROX_OPERATOR_NAMESPACE: ${STACKROX_OPERATOR_NAMESPACE}
INSTALL_OPENSHIFT_ROUTER: ${INSTALL_OPENSHIFT_ROUTER}
OCM_SERVICE_CLIENT_ID: ********
OCM_SERVICE_CLIENT_SECRET: ********
OCM_SERVICE_TOKEN: ********
SENTRY_KEY: ********
AWS_ACCESS_KEY: ********
AWS_ACCOUNT_ID: ********
AWS_SECRET_ACCESS_KEY: ********
SSO_CLIENT_ID: ********
SSO_CLIENT_SECRET: ********
OSD_IDP_SSO_CLIENT_ID: ********
OSD_IDP_SSO_CLIENT_SECRET: ********
ROUTE53_ACCESS_KEY: ********
ROUTE53_SECRET_ACCESS_KEY: ********
OBSERVABILITY_CONFIG_ACCESS_TOKEN: ********
INHERIT_IMAGEPULLSECRETS: ${INHERIT_IMAGEPULLSECRETS}
SPAWN_LOGGER: ${SPAWN_LOGGER}
DUMP_LOGS: ${DUMP_LOGS}
OPERATOR_SOURCE: ${OPERATOR_SOURCE}
INSTALL_OLM: ${INSTALL_OLM}
ENABLE_DB_PORT_FORWARDING: ${ENABLE_DB_PORT_FORWARDING}
ENABLE_FM_PORT_FORWARDING: ${ENABLE_FM_PORT_FORWARDING}
FLEETSHARD_SYNC_RESOURCES: ${FLEETSHARD_SYNC_RESOURCES}
DOCKER_CONFIG: ${DOCKER_CONFIG}
SKIP_TESTS: ${SKIP_TESTS}
ENABLE_CENTRAL_EXTERNAL_CERTIFICATE: ${ENABLE_CENTRAL_EXTERNAL_CERTIFICATE}
FLEET_MANAGER_IMAGE: ${FLEET_MANAGER_IMAGE}
FLEETSHARD_SYNC_CONTAINER_COMMAND: ${FLEETSHARD_SYNC_CONTAINER_COMMAND}
PATH: ${PATH}
EOF
}

wait_for_container_to_appear() {
    local namespace="$1"
    local pod_selector="$2"
    local container_name="$3"
    local seconds="${4:-150}" # Default to 150 seconds waiting time.

    log "Waiting for container ${container_name} within pod ${pod_selector} in namespace ${namespace} to appear..."
    for _ in $(seq "$seconds"); do
        local status
        status=$($KUBECTL -n "$namespace" get pod -l "$pod_selector" -o jsonpath="{.items[0].status.initContainerStatuses[?(@.name == '${container_name}')]} {.items[0].status.containerStatuses[?(@.name == '${container_name}')]}" 2>/dev/null || true)
        local state=""
        state=$(echo "${status}" | jq -r ".state | keys[]")
        if [[ "$state" == "running" ]]; then
            echo "Container ${pod_selector}/${container_name} is in state ${state}"
            return 0
        fi
        sleep 1
    done

    log "Timed out waiting for container ${container_name} to appear for pod ${pod_selector} in namespace ${namespace}"
    return 1
}

is_pod_ready() {
    local namespace="$1"
    local pod_selector="$2"
    local status
    status=$($KUBECTL -n "$namespace" get pod -l "$pod_selector" -o jsonpath="{.items[0].status.conditions[?(@.type == 'ContainersReady')].status}" 2>/dev/null || true)
    if [[ "$status" == "True" ]]; then
        return 0
    else
        return 1
    fi
}

wait_for_container_to_become_ready() {
    local namespace="$1"
    local pod_selector="$2"
    local container_name="$3"
    local timeout="${4:-300}s"

    log "Waiting for pod ${pod_selector} within namespace ${namespace} to become ready..."
    wait_for_container_to_appear "$namespace" "$pod_selector" "$container_name" || return 1

    $KUBECTL -n "$namespace" wait --timeout="$timeout" --for=condition=ready pod -l "$pod_selector"
    local exit_code="$?"
    if [[ exit_code -eq 0 ]]; then
        log "Container ${container_name} within namespace ${namespace} is ready."
        sleep 2
        return 0
    fi

    log "Failed to wait for container ${container_name} in pod ${pod_selector} in namespace ${namespace} to become ready"
    return 1
}

wait_for_resource_to_appear() {
    local namespace="$1"
    local kind="$2"
    local name="$3"
    local seconds="${4:-60}"

    log "Waiting for ${kind}/${name} to be created in namespace ${namespace}"

    for _ in $(seq "$seconds"); do
        if $KUBECTL -n "$namespace" get "$kind" "$name" 2>/dev/null >&2; then
            log "Resource ${kind}/${namespace} in namespace ${namespace} appeared"
            return 0
        fi
        sleep 1
    done

    log "Giving up after ${seconds}s waiting for ${kind}/${name} in namespace ${namespace}"

    return 1
}

wait_for_default_service_account() {
    local namespace="$1"
    if wait_for_resource_to_appear "$namespace" "serviceaccount" "default"; then
        return 0
    else
        return 1
    fi
}

assemble_kubeconfig() {
    kubeconf=$($KUBECTL config view --minify=true --raw=true 2>/dev/null)
    CONTEXT_NAME=$(echo "$kubeconf" | yq e .current-context -)
    CONTEXT="$(echo "$kubeconf" | yq e ".contexts[] | select(.name == \"${CONTEXT_NAME}\")" -o=json - | jq -c)"
    USER_NAME=$(echo "$CONTEXT" | jq -r .context.user -)
    CLUSTER_NAME=$(echo "$CONTEXT" | jq -r .context.cluster -)
    NEW_CONTEXT_NAME="$CLUSTER_NAME"
    CONTEXT=$(echo "$CONTEXT" | jq ".name = \"$NEW_CONTEXT_NAME\"" -c -)
    KUBEUSER="$(echo "$kubeconf" | yq e ".users[] | select(.name == \"${USER_NAME}\")" -o=json - | jq -c)"

    config=$(
        cat <<EOF
apiVersion: v1
clusters:
    - cluster:
        server: kubernetes.default.svc
      name: \"$CLUSTER_NAME\"
contexts:
    - $CONTEXT
current-context: "$NEW_CONTEXT_NAME"
kind: Config
users:
    - $KUBEUSER
EOF
    )

    echo "$config"
}

inject_ips() {
    local namespace="$1"
    local service_account="$2"
    local secret_name="$3"

    log "Patching ServiceAccount ${namespace}/${service_account} to use Quay.io imagePullSecrets"
    $KUBECTL -n "$namespace" patch sa "$service_account" -p "\"imagePullSecrets\": [{\"name\": \"${secret_name}\" }]"
}

is_local_cluster() {
    local cluster_type=${1:-}
    if [[ "$cluster_type" == "minikube" || "$cluster_type" == "colima" || "$cluster_type" == "rancher-desktop" || "$cluster_type" == "docker" || "$cluster_type" == "kind" ]]; then
        return 0
    else
        return 1
    fi
}

is_openshift_cluster() {
    local openshift_cluster_types="openshift,openshift-ci,crc,infra-openshift"
    local cluster_type="$1"
    if [[ -z "$cluster_type" ]]; then
        return 1
    fi
    if [[ ",${openshift_cluster_types}," == *",${cluster_type},"* ]]; then
        return 0
    else
        return 1
    fi
}

delete_tenant_namespaces() {
    central_namespaces=$($KUBECTL get namespace -o jsonpath='{range .items[?(@.status.phase == "Active")]}{.metadata.name}{"\n"}{end}' | grep '^rhacs-.*$' || true)
    if [[ ! "$central_namespaces" ]]; then
        log "No left-over RHACS tenant namespaces to be deleted."
        return
    fi
    for namespace in $central_namespaces; do
        $KUBECTL delete -n "$namespace" centrals.platform.stackrox.io --all || true
        $KUBECTL delete namespace "$namespace" &
    done
    log "Waiting for leftover RHACS namespaces to be deleted... "
    while true; do
        central_namespaces=$($KUBECTL get namespace -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' | grep '^rhacs-.*$' || true)
        if [[ "$central_namespaces" ]]; then
            central_namespaces_short=$(echo "$central_namespaces" | tr '\n' " ")
            log "Waiting for RHACS tenant namespaces to be deleted: $central_namespaces_short ..."
        else
            break
        fi
        sleep 1
    done
    log "All RHACS tenant namespaces deleted."
}
