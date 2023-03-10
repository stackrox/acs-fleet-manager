#!/usr/bin/env bash
set -eo pipefail

# This script is a small canary upgrade implementation for the ACS operator.
# It annotates all instances with a pause-reconcile annotation, upgrades the ACS operator.
# Afterwards the user can decide which instances to upgrade.

# upgrade_operator will pause all central instances and approve an install plan to upgrade the operator via OLM.
# The script asks for confirmation before upgrading.
# An upgrade is executed by unpausing a Central instance. This must be done manually.
upgrade_operator() {
    echo "Starting Operator upgrade (this will pause reconciliation on all Centrals) [yes/no]? "
    read -r answer_upgrade
    if [[ "$answer_upgrade" != "yes" ]]; then
        echo "User aborted."
        exit 1
    fi

    pause_all_centrals

    list_paused_centrals

    echo "Start executing Operator upgrade"

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
    read -r approve_install_answer
    if [[ "$approve_install_answer" != "yes" ]]; then
        echo "Aborting operator upgrade. User interrupt."
        exit 1
    fi

    echo "To confirm installation type \"$install_plan_version\":"
    read -r confirm_install_answer
    if [[ "$confirm_install_answer" = "$install_plan_version" ]]; then
        echo "Upgrade confirmed by user."
    else
        echo "Aborting operator upgrade. User interrupt, not confirmed."
        exit 1
    fi

    kubectl patch installplan -n "$install_plan_namespace" "$install_plan_name" -p '{"spec":{"approved": true}}' --type merge

    echo "Operator upgraded. Please check Operator health manually."
    sleep 1
    echo "Central instance upgrades must be performed manually by unpausing instances to be reconciled by the new operator version."
}

# pause_all_centrals sets all "stackrox.io/pause-reconcile" annotations to "true"
pause_all_centrals() {
    central_namespaces=$(get_central_namespaces)
    for ns in $central_namespaces
    do
      central_name=$(kubectl get centrals.platform.stackrox.io -n "$ns" | awk '(NR>1) {print $1}')
      kubectl -n "$ns" annotate centrals.platform.stackrox.io "$central_name" stackrox.io/pause-reconcile=true --overwrite=true
      echo "Paused: $central_name"
    done
}

# unpause_all_centrals sets all "stackrox.io/pause-reconcile" annotations to "false"
unpause_all_centrals() {
    central_namespaces=$(get_central_namespaces)
    for ns in $central_namespaces
    do
        central_name=$(kubectl get centrals.platform.stackrox.io -n "$ns" | awk '(NR>1) {print $1}')
        kubectl -n "$ns" annotate centrals.platform.stackrox.io "$central_name" stackrox.io/pause-reconcile=false --overwrite=true
        echo "Unpaused: $central_name"
    done
}

# list_paused_centrals lists all Central instances in a cluster which are annotated as "stackrox.io/pause-reconcile=true"
list_paused_centrals() {
    central_namespaces=$(get_central_namespaces)
    central_count=$(echo "$central_namespaces" | wc -l)

    paused_counter=0
    for ns in $central_namespaces
    do
      central=$(kubectl get centrals.platform.stackrox.io -n "$ns" -o json | jq '.items[0]')
      if [[ "$central" == "null" ]]; then
        continue
      fi

      is_paused=$(echo "$central" | jq '.metadata.annotations["stackrox.io/pause-reconcile"]' -r)
      if [[ "$is_paused" != "true" ]]; then
          continue
      fi

      echo "$central" | jq '{namespace: .metadata.namespace, name: .metadata.name}' -r
      paused_counter=$((paused_counter+1))
    done
    if [[ "$paused_counter" != "0" ]]; then
        echo ""
    fi

    central_count=$(echo "$central_count" | xargs) #remove whitespace
    echo "$paused_counter/$central_count Central instances are paused."
}

get_central_namespaces() {
    central_namespaces=$(kubectl get centrals.platform.stackrox.io -A | awk '(NR>1) {print $1}')
    if [[ -z "$central_namespaces" ]]; then
        echo "no central instance found"
        exit 1;
    fi
    echo "$central_namespaces"
}

case $1 in
  upgrade_operator)
    shift
    upgrade_operator "$@"
    ;;
  unpause_all_centrals)
    shift
    unpause_all_centrals "$@"
    ;;
  pause_all_centrals)
    shift
    pause_all_centrals "$@"
    ;;
  list_paused_centrals)
    shift
    list_paused_centrals "$@"
    ;;
  *)
    echo "unknown argument $1"
    exit 1
    ;;
esac
