die() {
    {
        printf "$*"
        echo
    } >&2
    exit 1
}

log() {
    printf "$*"
    echo
}

verify_environment() {
    return
}

get_current_cluster_name() {
    local kubeconfig_file="$1"
    local cluster_name=$($KUBECTL --kubeconfig "${kubeconfig_file}" config view --minify=true 2>/dev/null | yq e '.clusters[].name' -)
    echo "$cluster_name"
}

pre_init_env=$(env)

dump_env() {
    current_env=$(env)
    diff -up <(echo "$pre_init_env") <(echo "$current_env") |
        tail +4 |
        grep '^+' |
        sed -e 's/^\+//;' |
        grep -v '\(_DEFAULT\|^_\)=' |
        sort
}

init() {
    export GITROOT=${GITROOT:-"$(git rev-parse --show-toplevel)"} # This makes it possible to execute init without having GITROOT initialized already.
    export DEBUG="${DEBUG:-false}"
    set -eu -o pipefail
    if [[ "$DEBUG" == "trace" ]]; then
        set -x
    fi

    source "${GITROOT}/dev/env/defaults/env"
    if ! which bootstrap.sh >/dev/null 2>&1; then
        export PATH="$GITROOT/dev/env/scripts:${PATH}"
    fi

    if [[ -n "$OPENSHIFT_CI" ]]; then
        export CLUSTER_TYPE_DEFAULT="openshift"
    fi

    export CLUSTER_TYPE="${CLUSTER_TYPE:-$CLUSTER_TYPE_DEFAULT}"
    source "${GITROOT}/dev/env/defaults/cluster-type-${CLUSTER_TYPE}/env"

    if [[ -n "$OPENSHIFT_CI" ]]; then
        source "${GITROOT}/dev/env/defaults/openshift-ci.env"
    fi

    export ACSMS_NAMESPACE="${ACSMS_NAMESPACE:-$ACSMS_NAMESPACE_DEFAULT}"
    export KUBECTL=${KUBECTL:-$KUBECTL_DEFAULT}
    export DOCKER=${DOCKER:-$DOCKER_DEFAULT}
    export IMAGE_REGISTRY="${IMAGE_REGISTRY:-$IMAGE_REGISTRY_DEFAULT}"
    export STACKROX_OPERATOR_VERSION="${STACKROX_OPERATOR_VERSION:-$STACKROX_OPERATOR_VERSION_DEFAULT}"
    export CENTRAL_VERSION="${CENTRAL_VERSION:-$CENTRAL_VERSION_DEFAULT}"
    export SCANNER_VERSION="${SCANNER_VERSION:-$SCANNER_VERSION_DEFAULT}"
    export STACKROX_OPERATOR_NAMESPACE="${STACKROX_OPERATOR_NAMESPACE:-$STACKROX_OPERATOR_NAMESPACE_DEFAULT}"
    export STACKROX_OPERATOR_IMAGE="${IMAGE_REGISTRY}/stackrox-operator:${STACKROX_OPERATOR_VERSION}"
    export STACKROX_OPERATOR_INDEX_IMAGE="${IMAGE_REGISTRY}/stackrox-operator-index:v${STACKROX_OPERATOR_VERSION}"
    export KUBECONFIG=${KUBECONFIG:-$KUBECONFIG_DEFAULT}
    export CLUSTER_NAME_DEFAULT=$(get_current_cluster_name "$KUBECONFIG")
    export CLUSTER_NAME=${CLUSTER_NAME:-$CLUSTER_NAME_DEFAULT}
    export OPENSHIFT_MARKETPLACE="${OPENSHIFT_MARKETPLACE:-$OPENSHIFT_MARKETPLACE_DEFAULT}"
    export INSTALL_OPERATOR="${INSTALL_OPERATOR:-$INSTALL_OPERATOR_DEFAULT}"
    export POSTGRES_DB=${POSTGRES_DB:-$POSTGRES_DB_DEFAULT}
    export POSTGRES_USER=${POSTGRES_USER:-$POSTGRES_USER_DEFAULT}
    export POSTGRES_PASSWORD=${POSTGRES_PASSWORD:-$POSTGRES_PASSWORD_DEFAULT}
    export DATABASE_HOST="db"
    export DATABASE_PORT="5432"
    export DATABASE_NAME="$POSTGRES_DB"
    export DATABASE_USER="$POSTGRES_USER"
    export DATABASE_PASSWORD="$POSTGRES_PASSWORD"
    export DATABASE_TLS_CERT=""
    export OCM_SERVICE_CLIENT_ID=${OCM_SERVICE_CLIENT_ID:-$OCM_SERVICE_CLIENT_ID_DEFAULT}
    export OCM_SERVICE_CLIENT_SECRET=${OCM_SERVICE_CLIENT_SECRET:-$OCM_SERVICE_CLIENT_SECRET_DEFAULT}
    export OCM_SERVICE_TOKEN=${OCM_SERVICE_TOKEN:-$OCM_SERVICE_TOKEN_DEFAULT}
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
    export IMAGE_PULL_DOCKER_CONFIG=${IMAGE_PULL_DOCKER_CONFIG:-$IMAGE_PULL_DOCKER_CONFIG_DEFAULT}
    export KUBECONF_CLUSTER_SERVER_OVERRIDE=${KUBECONF_CLUSTER_SERVER_OVERRIDE:-$KUBECONF_CLUSTER_SERVER_OVERRIDE_DEFAULT}
    export INHERIT_IMAGEPULLSECRETS=${INHERIT_IMAGEPULLSECRETS:-$INHERIT_IMAGEPULLSECRETS_DEFAULT}
    export SPAWN_LOGGER=${SPAWN_LOGGER:-$SPAWN_LOGGER_DEFAULT}
    export DUMP_LOGS=${DUMP_LOGS:-$DUMP_LOGS_DEFAULT}
    export OPERATOR_SOURCE=${OPERATOR_SOURCE:-$OPERATOR_SOURCE_DEFAULT}
    export INSTALL_OLM=${INSTALL_OLM:-$INSTALL_OLM_DEFAULT}
    export ENABLE_DB_PORT_FORWARDING=${ENABLE_DB_PORT_FORWARDING:-$ENABLE_DB_PORT_FORWARDING_DEFAULT}
    export ENABLE_FM_PORT_FORWARDING=${ENABLE_FM_PORT_FORWARDING:-$ENABLE_FM_PORT_FORWARDING_DEFAULT}
    export AUTH_TYPE=${AUTH_TYPE:-$AUTH_TYPE_DEFAULT}
    export FINAL_TEAR_DOWN=${FINAL_TEAR_DOWN:-$FINAL_TEAR_DOWN_DEFAULT}
    export FLEET_MANAGER_RESOURCES=${FLEET_MANAGER_RESOURCES:-$FLEET_MANAGER_RESOURCES_DEFAULT}
    export FLEETSHARD_SYNC_RESOURCES=${FLEETSHARD_SYNC_RESOURCES:-$FLEETSHARD_SYNC_RESOURCES_DEFAULT}
    export DB_RESOURCES=${DB_RESOURCES_DEFAULT:-$DB_RESOURCES_DEFAULT}
    export RHACS_OPERATOR_RESOURCES=${RHACS_OPERATOR_RESOURCES:-$RHACS_OPERATOR_RESOURCES_DEFAULTS}

    export FLEET_MANAGER_IMAGE="${FLEET_MANAGER_IMAGE:-$FLEET_MANAGER_IMAGE_DEFAULT}"
    # When transferring images without repository hostname to Minikube it gets prefixed with "docker.io" automatically.
    if [[ "$FLEET_MANAGER_IMAGE" =~ ^fleet-manager-.*/fleet-manager:.* ]]; then
        export FULL_FLEET_MANAGER_IMAGE="docker.io/${FLEET_MANAGER_IMAGE}"
    else
        export FULL_FLEET_MANAGER_IMAGE="${FLEET_MANAGER_IMAGE}"
    fi

    verify_environment

    disable_debugging
    enable_debugging_if_necessary
}

disable_debugging() {
    if [[ "$DEBUG" != "trace" ]]; then
        set +x
    fi
}

enable_debugging_if_necessary() {
    if [[ "$DEBUG" != "false" ]]; then
        set -x
    fi
}

wait_for_container_to_appear() {
    local namespace="$1"
    local pod_selector="$2"
    local container_name="$3"

    log "Waiting for container ${container_name} within pod ${pod_selector} in namespace ${namespace} to appear..."
    for i in $(seq 60); do
        local status=$($KUBECTL -n "$ACSMS_NAMESPACE" get pod -l "${pod_selector}" -o jsonpath="{.items[0].status.initContainerStatuses[?(@.name == '${container_name}')]} {.items[0].status.containerStatuses[?(@.name == '${container_name}')]}" 2>/dev/null)
        local state=$(echo "${status}" | jq -r ".state | keys[]")
        if [[ "$state" == "terminated" || "$state" == "running" ]]; then
            echo "Container ${container_name} is in state ${state}"
            sleep 2
            break
        fi
        sleep 2
    done
}

wait_for_container_to_become_ready() {
    local namespace="$1"
    local pod_selector="$2"
    local container="$3"

    log "Waiting for pod ${pod_selector} within namespace ${namespace} to become ready..."
    wait_for_container_to_appear "$namespace" "$pod_selector" "$container"
    for i in $(seq 10); do
        if $KUBECTL -n "$namespace" wait --timeout=5s --for=condition=ready pod -l "$pod_selector" 2>/dev/null >&2; then
            break
        fi
        sleep 2
    done
    $KUBECTL -n "$namespace" wait --timeout=60s --for=condition=ready pod -l "$pod_selector"
    sleep 2
    log "Pod ${pod_selector} is ready."
}

wait_for_resource_to_appear() {
    local namespace="$1"
    local kind="$2"
    local name="$3"

    log "Waiting for ${kind}/${name} to be created in namespace ${namespace}"

    for i in $(seq 60); do
        if $KUBECTL -n "$namespace" get "$kind" "$name" 2>/dev/null >&2; then
            return 0
        fi
        sleep 1
    done

    log "Giving up waiting for ${kind}/${name} in namespace ${namespace}"

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
