#!/bin/bash
set -eo pipefail

if [[ $# -ne 2 ]]; then
    echo "Usage: $0 [deployment name] [patch]" >&2
    echo "Note: you need to be logged into OC for your cluster's administrator"
    exit 2
fi

deployment_name=$1
patch=$2

rhacs_namespaces=$(oc get ns | grep -E "^rhacs-[a-z0-9]{20}" | awk '{print $1}')

for namespace in $rhacs_namespaces;
do
  oc -n "${namespace}" patch deploy/"${deployment_name}" -p "${patch}"
done
