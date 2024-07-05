#!/usr/bin/env bash

# This script creates a kubernetes pull secret for pulling Red Hat images
# Usage: ./scripts/redhat-pull-secret.sh <namespace1> <namespace2> ...

set -eou pipefail

echo "Creating Red Hat pull secret in namespaces" "$@"

token=$(ocm token)
pull_secret=$(curl -X POST https://api.openshift.com/api/accounts_mgmt/v1/access_token \
    --header "Content-Type:application/json" \
    --header "Authorization: Bearer ${token}")

for namespace in "$@"; do

    # Create namespace if it does not exist
    oc get namespace "$namespace" || oc create namespace "$namespace"

    # Wait for namespace to be created
    trial=0
    while [ "$(oc get namespace "$namespace" -o jsonpath='{.status.phase}')" != "Active" ]; do
        echo "Waiting for namespace $namespace to be created"
        trial=$((trial + 1))
        if [ "$trial" -gt 10 ]; then
            echo "Timeout waiting for namespace $namespace to be created"
            exit 1
        fi
        sleep 5
    done

    echo "Creating RedHat Pull secret in namespace $namespace"
    cat <<EOF | oc apply -n "$namespace" -f-
kind: Secret
apiVersion: v1
type: kubernetes.io/dockerconfigjson
metadata:
  name: redhat-pull-secret
data:
    .dockerconfigjson: $(echo "${pull_secret}" | jq '. | @base64')
EOF
    echo "RedHat Pull Secret created in namespace $namespace"

done
