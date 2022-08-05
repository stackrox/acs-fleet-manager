#!/usr/bin/env bash

set -eo pipefail

SCRIPT_DIR=$(realpath "$(dirname "${BASH_SOURCE[0]}")")

ROUTER_RBAC_YAML="https://raw.githubusercontent.com/openshift/router/master/deploy/router_rbac.yaml"
ROUTER_CRD_YAML="https://raw.githubusercontent.com/openshift/api/master/route/v1/route.crd.yaml"
ROUTER_DEPLOYMENT_YAML="https://raw.githubusercontent.com/openshift/router/master/deploy/router.yaml"
APPS_OPENSHIFT_CRD_YAML="$SCRIPT_DIR/../dev/env/manifests/ingress-router/dummy.crd.yaml"

usage() {
  echo "Usage: $(basename "$0") [-h | --help] <command> [<args>]"
  echo ""
  echo "Available commands:"
  echo "  deploy               Deploys ingress router on an active k8s cluster in kubectl context"
  echo "  undeploy             Un-deploys ingress router on an active k8s cluster in kubectl context"
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

POSITIONAL_ARGS=()

while (("$#")); do
  case "$1" in
  -h | --help)
    usage
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
*)
  usage "Unknown command: $command"
  ;;
esac
