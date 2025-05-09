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
    export STACKROX_OPERATOR_NAMESPACE="${STACKROX_OPERATOR_NAMESPACE:-$STACKROX_OPERATOR_NAMESPACE_DEFAULT}"
    export INSTALL_OPENSHIFT_ROUTER="${INSTALL_OPENSHIFT_ROUTER:-$INSTALL_OPENSHIFT_ROUTER_DEFAULT}"
    export EXPOSE_OPENSHIFT_ROUTER="${EXPOSE_OPENSHIFT_ROUTER:-$EXPOSE_OPENSHIFT_ROUTER_DEFAULT}"
    export INSTALL_VERTICAL_POD_AUTOSCALER="${INSTALL_VERTICAL_POD_AUTOSCALER:-$INSTALL_VERTICAL_POD_AUTOSCALER_DEFAULT}"
    export INSTALL_VERTICAL_POD_AUTOSCALER_OLM="${INSTALL_VERTICAL_POD_AUTOSCALER_OLM:-$INSTALL_VERTICAL_POD_AUTOSCALER_OLM_DEFAULT}"
    export INSTALL_EXTERNAL_SECRETS="${INSTALL_EXTERNAL_SECRETS:-$INSTALL_EXTERNAL_SECRETS_DEFAULT}"
    export EXTERNAL_SECRETS_VERSION="${EXTERNAL_SECRETS_VERSION:-$EXTERNAL_SECRETS_VERSION_DEFAULT}"
    export OCM_SERVICE_CLIENT_ID=${OCM_SERVICE_CLIENT_ID:-$OCM_SERVICE_CLIENT_ID_DEFAULT}
    export OCM_SERVICE_CLIENT_SECRET=${OCM_SERVICE_CLIENT_SECRET:-$OCM_SERVICE_CLIENT_SECRET_DEFAULT}
    export OCM_SERVICE_TOKEN=${OCM_SERVICE_TOKEN:-$OCM_SERVICE_TOKEN_DEFAULT}
    export OCM_ADDON_SERVICE_CLIENT_ID=${OCM_ADDON_SERVICE_CLIENT_ID:-$OCM_ADDON_SERVICE_CLIENT_ID_DEFAULT}
    export OCM_ADDON_SERVICE_CLIENT_SECRET=${OCM_ADDON_SERVICE_CLIENT_SECRET:-$OCM_ADDON_SERVICE_CLIENT_SECRET_DEFAULT}
    export OCM_ADDON_SERVICE_TOKEN=${OCM_ADDON_SERVICE_TOKEN:-$OCM_ADDON_SERVICE_TOKEN_DEFAULT}
    export ROUTE53_ACCESS_KEY=${ROUTE53_ACCESS_KEY:-$ROUTE53_ACCESS_KEY_DEFAULT}
    export ROUTE53_SECRET_ACCESS_KEY=${ROUTE53_SECRET_ACCESS_KEY:-$ROUTE53_SECRET_ACCESS_KEY_DEFAULT}
    export SPAWN_LOGGER=${SPAWN_LOGGER:-$SPAWN_LOGGER_DEFAULT}
    export DUMP_LOGS=${DUMP_LOGS:-$DUMP_LOGS_DEFAULT}
    export ENABLE_DB_PORT_FORWARDING=${ENABLE_DB_PORT_FORWARDING:-$ENABLE_DB_PORT_FORWARDING_DEFAULT}
    export ENABLE_FM_PORT_FORWARDING=${ENABLE_FM_PORT_FORWARDING:-$ENABLE_FM_PORT_FORWARDING_DEFAULT}
    export FLEETSHARD_SYNC_RESOURCES=${FLEETSHARD_SYNC_RESOURCES:-$FLEETSHARD_SYNC_RESOURCES_DEFAULT}
    export SKIP_TESTS=${SKIP_TESTS:-$SKIP_TESTS_DEFAULT}
    export ENABLE_CENTRAL_EXTERNAL_DOMAIN=${ENABLE_CENTRAL_EXTERNAL_DOMAIN:-$ENABLE_CENTRAL_EXTERNAL_DOMAIN_DEFAULT}
    export FLEET_MANAGER_IMAGE=${FLEET_MANAGER_IMAGE:-$FLEET_MANAGER_IMAGE_DEFAULT}
    export ENABLE_EMAIL_SENDER=${ENABLE_EMAIL_SENDER:-$ENABLE_EMAIL_SENDER_DEFAULT}
    export EMAIL_SENDER_IMAGE=${EMAIL_SENDER_IMAGE:-$EMAIL_SENDER_IMAGE_DEFAULT}
    export EMAIL_SENDER_RESOURCES=${EMAIL_SENDER_RESOURCES:-$EMAIL_SENDER_RESOURCES_DEFAULT}
    export MANAGED_DB_ENABLED=${MANAGED_DB_ENABLED:-$MANAGED_DB_ENABLED_DEFAULT}
    export ARGOCD_TENANT_APP_TARGET_REVISION=${ARGOCD_TENANT_APP_TARGET_REVISION:-$ARGOCD_TENANT_APP_TARGET_REVISION_DEFAULT}

    FLEETSHARD_SYNC_CONTAINER_COMMAND_DEFAULT="/usr/local/bin/fleetshard-sync"
    export FLEETSHARD_SYNC_CONTAINER_COMMAND=${FLEETSHARD_SYNC_CONTAINER_COMMAND:-$FLEETSHARD_SYNC_CONTAINER_COMMAND_DEFAULT}

    if [[ "$FLEET_MANAGER_IMAGE" == "" ]]; then
        FLEET_MANAGER_IMAGE=$(make -s -C "$GITROOT" full-image-tag)
        log "FLEET_MANAGER_IMAGE not set, using ${FLEET_MANAGER_IMAGE}"
    fi

    if [[ "$ENABLE_CENTRAL_EXTERNAL_DOMAIN" != "false" && ("$ROUTE53_ACCESS_KEY" == "" || "$ROUTE53_SECRET_ACCESS_KEY" == "") ]]; then
        log "setting ENABLE_CENTRAL_EXTERNAL_DOMAIN to false since no Route53 credentials were provided"
        ENABLE_CENTRAL_EXTERNAL_DOMAIN=false
    fi

    if [[ "$CLUSTER_TYPE" == "minikube" ]]; then
        eval "$(minikube docker-env)"
    fi
    if [[ "$CLUSTER_TYPE" == "crc" ]]; then
        eval "$(crc podman-env)"
    fi

    if [[ "$EMAIL_SENDER_IMAGE" == "" ]]; then
        EMAIL_SENDER_IMAGE=$(make -s -C "$GITROOT" image-tag/emailsender)
        log "EMAIL_SENDER_IMAGE was not set, use ${EMAIL_SENDER_IMAGE}"
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
STACKROX_OPERATOR_NAMESPACE: ${STACKROX_OPERATOR_NAMESPACE}
INSTALL_OPENSHIFT_ROUTER: ${INSTALL_OPENSHIFT_ROUTER}
EXPOSE_OPENSHIFT_ROUTER: ${EXPOSE_OPENSHIFT_ROUTER}
INSTALL_VERTICAL_POD_AUTOSCALER: ${INSTALL_VERTICAL_POD_AUTOSCALER}
INSTALL_VERTICAL_POD_AUTOSCALER_OLM: ${INSTALL_VERTICAL_POD_AUTOSCALER_OLM}
INSTALL_ARGOCD: ${INSTALL_ARGOCD}
INSTALL_OPENSHIFT_GITOPS: ${INSTALL_OPENSHIFT_GITOPS}
INSTALL_EXTERNAL_SECRETS: ${INSTALL_EXTERNAL_SECRETS}
EXTERNAL_SECRETS_VERSION: ${EXTERNAL_SECRETS_VERSION}
ARGOCD_NAMESPACE: ${ARGOCD_NAMESPACE}
ARGOCD_TENANT_APP_TARGET_REVISION: ${ARGOCD_TENANT_APP_TARGET_REVISION}
OCM_SERVICE_CLIENT_ID: ********
OCM_SERVICE_CLIENT_SECRET: ********
OCM_SERVICE_TOKEN: ********
ROUTE53_ACCESS_KEY: ********
ROUTE53_SECRET_ACCESS_KEY: ********
SPAWN_LOGGER: ${SPAWN_LOGGER}
DUMP_LOGS: ${DUMP_LOGS}
ENABLE_DB_PORT_FORWARDING: ${ENABLE_DB_PORT_FORWARDING}
ENABLE_FM_PORT_FORWARDING: ${ENABLE_FM_PORT_FORWARDING}
FLEETSHARD_SYNC_RESOURCES: ${FLEETSHARD_SYNC_RESOURCES}
SKIP_TESTS: ${SKIP_TESTS}
ENABLE_CENTRAL_EXTERNAL_DOMAIN: ${ENABLE_CENTRAL_EXTERNAL_DOMAIN}
FLEET_MANAGER_IMAGE: ${FLEET_MANAGER_IMAGE}
FLEETSHARD_SYNC_CONTAINER_COMMAND: ${FLEETSHARD_SYNC_CONTAINER_COMMAND}
EMAIL_SENDER_IMAGE: ${EMAIL_SENDER_IMAGE}
EMAIL_SENDER_RESOURCES: ${EMAIL_SENDER_RESOURCES}
PATH: ${PATH}
EOF
}

wait_for_img() {
    local img="$1"
    local seconds="${2:-1200}"
    local backoff=30

    local start
    start="$(date +%s)"
    local time_diff=0

    echo "Waiting for $img to become available..."
    while [ "$time_diff" -le "$seconds" ]
    do
        # the grep depends on the error message docker prints if manifest is not found
        if ! docker manifest inspect "$img" 2>&1 | grep "no such manifest"; then
            echo "remote image: $img is available."
            return 0
        fi

        local unix_seconds_now
        unix_seconds_now="$(date +%s)"
        time_diff=$(( unix_seconds_now - start))
        sleep "$backoff"
    done

    echo "Timed out waiting for remote image: $img to become available"
    return 1
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
            log "Resource ${kind}/${name} in namespace ${namespace} appeared"
            return 0
        fi
        sleep 1
    done

    log "Giving up after ${seconds}s waiting for ${kind}/${name} in namespace ${namespace}"

    return 1
}

wait_for_cluster_resource_to_appear() {
    local kind="$1"
    local name="$2"
    local seconds="${3:-60}"

    log "Waiting for ${kind}/${name} to be created"
    for _ in $(seq "$seconds"); do
        if $KUBECTL get "$kind" "$name" 2>/dev/null >&2; then
            log "Resource ${kind}/${name} appeared"
            return 0
        fi
        sleep 1
    done

    log "Giving up after ${seconds}s waiting for ${kind}/${name}"
    return 1
}


wait_for_resource_condition() {
    local namespace="$1"
    local kind="$2"
    local name="$3"
    local condition="$4"

    if ! wait_for_resource_to_appear "${namespace}" "${kind}" "${name}"; then
        return 1
    fi
    log "Waiting for ${kind}/${name} in namespace ${namespace} to have status ${condition}"
    $KUBECTL -n "${namespace}" wait --for="${condition}" --timeout="3m" "${kind}/${name}"
}

wait_for_cluster_resource_condition() {
    local kind="$1"
    local name="$2"
    local condition="$3"

    if ! wait_for_cluster_resource_to_appear "${kind}" "${name}"; then
        return 1
    fi
    log "Waiting for ${kind}/${name} to have status ${condition}"
    $KUBECTL wait --for "${condition}" --timeout="3m" "${kind}/${name}"
}

wait_for_crd() {
    local name="$1"
    wait_for_cluster_resource_condition crd "$name" "condition=established"
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
    # Filter regex is based on https://github.com/rs/xid
    central_namespaces=$($KUBECTL get namespace -o jsonpath='{range .items[?(@.status.phase == "Active")]}{.metadata.name}{"\n"}{end}' | grep -E '^rhacs-[0-9a-v]{20}$' || true)
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
        # Filter regex is based on https://github.com/rs/xid
        central_namespaces=$($KUBECTL get namespace -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' | grep -E '^rhacs-[0-9a-v]{20}$' || true)
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

print_pull_secret() {
    local name="$1"
    [[ -n "$name" ]] || die "Image pull secret name is empty"
    local registry_auth="$2"
    [[ -n "$registry_auth" ]] || die "Unable to create an image pull secret with name $name: .dockerconfigjson is empty"
    cat <<EOF
apiVersion: v1
data:
  .dockerconfigjson: ${registry_auth}
kind: Secret
metadata:
  name: $name
type: kubernetes.io/dockerconfigjson
EOF
}

redhat_registry_auth() {
    # Try to fetch an access token from ocm.
    local registry_auth
    registry_auth=$(ocm post /api/accounts_mgmt/v1/access_token <<< '' 2>/dev/null | jq -r '. | @base64')
    if [ -n "$registry_auth" ]; then
        echo "$registry_auth"
        return
    fi
    # If failed, fallback to retrieving credentials from docker config / cred store.
    docker_auth.sh -m k8s registry.redhat.io
}

quay_registry_auth() {
    REGISTRY_USERNAME="${QUAY_USER:-}" REGISTRY_PASSWORD="${QUAY_TOKEN:-}" docker_auth.sh -m k8s quay.io
}

# support both registry.redhat.io and quay.io to quickly switch images between upstream and downstream.
# order is important, the latter takes precedence (overrides) in case the registry is defined in both auth-s
composite_registry_auth() {
    echo "$(redhat_registry_auth | base64 -d)" "$(quay_registry_auth | base64 -d)" | jq -s -r 'reduce .[] as $x ({}; . * $x) | @base64'
}
