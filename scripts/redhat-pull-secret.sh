#!/usr/bin/env bash

# This script creates a kubernetes pull secret for pulling Red Hat images
# Usage: ./scripts/redhat-pull-secret.sh <namespace1> <namespace2> ...

set -eou pipefail

token=$(ocm token)
pull_secret=$(curl -X POST https://api.openshift.com/api/accounts_mgmt/v1/access_token \
    --header "Content-Type:application/json" \
    --header "Authorization: Bearer ${token}")

for namespace in "$@"; do

    # Create namespace if it does not exist
    oc get namespace "$namespace" || oc create namespace "$namespace"

    echo "Creating secret in namespace $namespace"
    cat <<EOF | oc apply -n "$namespace" -f -
kind: Secret
apiVersion: v1
type: kubernetes.io/dockerconfigjson
metadata:
  name: redhat-pull-secret
data:
    .dockerconfigjson: $(echo "$pull_secret" | base64)
EOF
done
