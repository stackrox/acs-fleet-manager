#!/usr/bin/env bash

set -eo pipefail

SCRIPT_DIR=$(realpath "$(dirname "${BASH_SOURCE[0]}")")

ROUTER_RBAC_YAML="https://raw.githubusercontent.com/openshift/router/master/deploy/router_rbac.yaml"
ROUTER_CRD_YAML="https://raw.githubusercontent.com/openshift/api/master/route/v1/route.crd.yaml"
ROUTER_DEPLOYMENT_YAML="https://raw.githubusercontent.com/openshift/router/master/deploy/router.yaml"
APPS_OPENSHIFT_CRD_YAML="$SCRIPT_DIR/../dev/env/manifests/ingress-router/dummy.crd.yaml"
HOSTCTL_PROFILE="acs"

usage() {
  echo "Usage: $(basename "$0") [-h | --help] <command> [<args>]"
  echo ""
  echo "Available commands:"
  echo "  deploy                                                          Deploys ingress router on an active k8s cluster in kubectl context (requires kubectl)"
  echo "  undeploy                                                        Un-deploys ingress router on an active k8s cluster in kubectl context (requires kubectl)"
  echo "  host                                                            Helper commands for exposing Routes locally"
  echo "                                                                  See more: https://github.com/stackrox/acs-fleet-manager/blob/main/docs/development/test-locally-route-hosts.md"
  echo "    host add (--profile=<profile> | -p <profile>) <hostname>      Adds the selected hostname to /etc/hosts (requires: kubectl, jq, hostctl, fzf). Profile: hostctl profile. Default: 'acs'"
  echo "    host remove (--profile=<profile> | -p <profile>) <hostname>   Removes the selected hostname from /etc/host (requires: kubectl, jq, hostctl, fzf). Profile: hostctl profile. Default: 'acs'"

  if [[ -n "${1:-}" ]]; then
    echo ""
    echo >&2 "Error: $1"
    exit 2
  fi
  exit 0
}

deploy() {
  for manifest in $(list_manifests); do
    kubectl create -f "$manifest"
  done
}

undeploy() {
    for manifest in $(list_manifests_reversed); do
      kubectl delete -f "$manifest" || true
    done
}

list_manifests() {
  echo -e "$ROUTER_RBAC_YAML\n$ROUTER_CRD_YAML\n$ROUTER_DEPLOYMENT_YAML\n$APPS_OPENSHIFT_CRD_YAML"
}

list_manifests_reversed() {
  list_manifests | tac
}

add_host() {
  local host context
  context=$(kubectl config current-context)
  host=$1
  if [[ -z $host ]]; then
    routes=$(kubectl get routes -l 'app.kubernetes.io/managed-by=rhacs-fleetshard' --all-namespaces -o json)
    host=$(jq -r '.items[].spec.host' <<< "$routes" | fzf --header "Using context $context")
  fi
  sudo hostctl add domains "$HOSTCTL_PROFILE" "$host"
}

remove_host() {
  local host=$1
  if [[ -z $host ]]; then
    host=$(hostctl list "$HOSTCTL_PROFILE" -o json | jq -r ".[].Host" | fzf)
  fi
  sudo hostctl remove domains "$HOSTCTL_PROFILE" "$host"
}

POSITIONAL_ARGS=()

while (("$#")); do
  case "$1" in
  -h | --help)
    usage
    ;;
  -p)
    shift
    HOSTCTL_PROFILE="$1"
    shift
    ;;
  --profile=*)
    HOSTCTL_PROFILE="${1#*=}"
    shift
    ;;
  -*)
    usage "Unknown option $1"
    ;;
  *)
    POSITIONAL_ARGS+=("$1")
    shift
    ;;
  esac
done

set -- "${POSITIONAL_ARGS[@]}"

command=$1
[[ -n $command ]] || usage "No command specified"

case $command in
deploy)
  deploy
  ;;
undeploy)
  undeploy
  ;;
host)
  subcommand=$2
  host=$3
  case $subcommand in
  add)
    add_host "$host"
    ;;
  remove)
    remove_host "$host"
    ;;
  *)
    usage "Unknown host command: $subcommand"
    ;;
  esac
  ;;
*)
  usage "Unknown command: $command"
  ;;
esac
