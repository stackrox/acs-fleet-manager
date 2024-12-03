#!/usr/bin/env bash

# This script assumes that there were two clusters created before execution.
# It expects that those clusters are accessible through kubeconfig files at the path
# value stored in following environment variables:
export CLUSTER_1_KUBECONFIG=${CLUSTER_1_KUBECONFIG:-"$HOME/.kube/cluster1"}
export CLUSTER_2_KUBECONFIG=${CLUSTER_2_KUBECONFIG:-"$HOME/.kube/cluster2"}
# During execution cluster 1 will act as control plane and data plane running fleet-manager and fleetshard components
# Cluster 2 will act as a data plane, running only fleetshard components

# Bootstrap C1
export KUBECONFIG="$CLUSTER_1_KUBECONFIG"
export INHERIT_IMAGEPULLSECRETS="true" # pragma: allowlist secret

# TODO: Double check how setup is done in OSCI so that we
# Get the propper certificates to allow enabling creation of routes and DNS entries
# Get the propper secrets to allow communication of FM to Route 53
# Get the quay configuration to pull images
# Maybe we wanna rely on prebuild images instead of building them ourselves, which might
# as well need additional / other commands
make deploy/bootstrap
make deploy/dev

# service template for dev defines a reencrypt route which requires manual creation of a self
# signed certificate before starting fleet-manager. We don't want that which is why we're seeting
# termination to edge for this route
kubectl patch -n rhacs route fleet-manager -p '{"spec":{"tls":{"termination":"edge"}}}'
FM_URL="https://$(k get routes -n rhacs fleet-manager -o yaml | yq .spec.host)"
export FM_URL

kubectl get cm -n rhacs fleet-manager-dataplane-cluster-scaling-config -o yaml > fm-dataplane-config.yaml
yq '.data."dataplane-cluster-configuration.yaml"' fm-dataplane-config.yaml | yq .clusters > cluster-list.json

KUBECONFIG="$CLUSTER_2_KUBECONFIG" make cluster-list \
  | jq '.[0] | .name="dev2" | .cluster_id="1234567890abcdef1234567890abcdeg"' \
  | jq --slurp . > cluster-list2.json

cluster_list_value=$(jq --slurp '. | add' cluster-list.json cluster-list2.json -c)
export new_cluster_config_value="clusters: $cluster_list_value"
yq -i '.data."dataplane-cluster-configuration.yaml" = strenv(new_cluster_config_value)' fm-dataplane-config.yaml

kubectl apply -f fm-dataplane-config.yaml
# Restart fleet-manager to pickup the config
kubectl delete pod -n rhacs -l app=fleet-manager

export KUBECONFIG="$CLUSTER_2_KUBECONFIG"
make deploy/bootstrap
make deploy/dev

# TODO: make a knob that allows for only deploying FS
# remove control plane deployment from 2nd cluster
kubectl delete deploy fleet-manager fleet-manager-db -n rhacs

# Get a static token from cluster1 which will be used for FS -> FM communication
# for the FS running on cluster2
STATIC_TOKEN=$(KUBECONFIG="$CLUSTER_1_KUBECONFIG" kubectl create token -n rhacs fleetshard-sync --audience acs-fleet-manager-private-api --duration 8760h)

# Configure FS on cluster2 to reach out to FM on cluster1
kubectl patch fleetshards -n rhacs rhacs-terraform --type='merge' -p "{\"spec\":{\"fleetshardSync\":{\"authType\":\"STATIC_TOKEN\",\"staticToken\":\"$STATIC_TOKEN\",\"fleetManagerEndpoint\":\"$FM_URL\",\"clusterId\":\"1234567890abcdef1234567890abcdeg\"}}}"

# TODO: remove this as soon as the feature flag RHACS_CLUSTER_MIGRATION is retired
export KUBECONFIG=$CLUSTER_1_KUBECONFIG
kubectl patch deploy -n rhacs fleetshard-sync -p '{"spec":{"template":{"spec":{"containers":[{"name":"fleetshard-sync","env":[{"name":"RHACS_CLUSTER_MIGRATION", "value":"true"}]}]}}}}'
kubectl patch deploy -n rhacs fleet-manager -p '{"spec":{"template":{"spec":{"containers":[{"name":"service","env":[{"name":"RHACS_CLUSTER_MIGRATION", "value":"true"}]}]}}}}'

export KUBECONFIG=$CLUSTER_2_KUBECONFIG
kubectl patch deploy -n rhacs fleetshard-sync -p '{"spec":{"template":{"spec":{"containers":[{"name":"fleetshard-sync","env":[{"name":"RHACS_CLUSTER_MIGRATION", "value":"true"}]}]}}}}'
# Start test execution in Go
