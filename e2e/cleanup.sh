#!/usr/bin/env bash
set -eo pipefail

namespaces=$(kubectl get ns | grep e2e-test-central | awk '{ print $1 }' | tr '\n' ' ')
for namespace in $namespaces;
do
  kubectl delete namespace "$namespace" &
done
make db/teardown
