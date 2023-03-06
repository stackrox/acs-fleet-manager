#!/usr/bin/env bash
set -exo pipefail

# This script is a small canary upgrade implementation for the ACS operator.
# It annotates all instances with a pause-reconcile annotation, upgrades the ACS operator.
# Afterwards the user can decide which instances to upgrade.

upgrade_operator() {
    central_namespaces=$(kubectl get centrals.platform.stackrox.io -A | awk '(NR>1) {print $1}')
    if [[ -z "$central_namespaces" ]]; then
        echo "no central instance found"
        exit 1;
    fi

    for ns in $central_namespaces
    do
        central_name=$(kubectl get centrals.platform.stackrox.io -n "$ns" | awk '(NR>1) {print $1}')
        kubectl -n "$ns" annotate centrals.platform.stackrox.io "$central_name" stackrox.io/pause-reconcile=true
        echo "Paused: $central_name"
    done

    echo "Executing Operator upgrade"

    # receive install plan
    install_plan=$(kubectl get installplans.operators.coreos.com -A -o json | jq '.items[] | select(.spec.approved == false) | select(.spec.clusterServiceVersionNames[])')
    install_plan_name=$(echo "$install_plan" | jq '.metadata.name' -r)
    install_plan_namespace=$(echo "$install_plan" | jq '.metadata.namespace' -r)
    install_plan_version=$(echo "$install_plan" | jq '.spec.clusterServiceVersionNames[0]' -r)

    # in case this fails some data could not be parsed with jq. Probably no install plan.
    if [[ "$install_plan_name" == "null" || "$install_plan_namespace" == "null" || "$install_plan_version" == "null" ]]; then
      echo "Could not parse install plan, missing data."
      exit 1
    fi

    echo "Approving installplan $install_plan_namespace/$install_plan_name to $install_plan_version? [yes/no]"
    read apporve_install_answer
    if [[ "$approve_install_answer" -ne "yes" ]]; then
        echo "Aborting operator upgrade. User interrupt."
        exit 1
    fi

    echo "To confirm installation type \""$install_plan_version"\":"
    read confirm_install_answer
    if [[ "$confirm_install_answer" = "$install_plan_version" ]]; then
        echo "Upgrade confirmed by user."
    else
        echo "Aborting operator upgrade. User interrupt, not confirmed."
        exit 1
    fi

    kubectl patch installplan -n "$install_plan_namespace" $install_plan_name -p '{"spec":{"approved": true}}' --type merge
}

## TODO: implement upgrade first canary
## TODO: implement upgrade all other instances
## TODO: list paused instances
upgrade_instance() {
    id="$1"
    echo $id
}

case $1 in
  upgrade_operator)
    shift
    upgrade_operator "$@"
    ;;
  upgrade_instance)
    shift
    upgrade_instance "$@"
    ;;
  upgrade_all)
    shift
    upgrade_instance "$@"
    ;;
  *)
    echo "unknown argument $1"
    exit 1
    ;;
esac
