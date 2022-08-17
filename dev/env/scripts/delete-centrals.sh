#!/usr/bin/env bash

# This script deletes all centrals in the database and destroys
# all Kubernetes resources associated with them.

set -e

KUBECTL=${KUBECTL:-kubectl}

psql -h localhost -U fleet_manager -d rhacsms -c "DELETE FROM central_requests;"

for namespace in $($KUBECTL get namespace -o 'jsonpath={.items[*].metadata.name}'); do
    if [[ "$namespace" =~ ^rhacs- ]]; then
        $KUBECTL delete namespace "$namespace"
    fi
done
