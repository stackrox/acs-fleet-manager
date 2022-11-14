#!/usr/bin/env bash
set -eo pipefail

# Approve the first unapproved install plan

installPlan=$(kubectl get installplan | grep false | awk '{print $1}')
echo "Updating install plan $installPlan"
kubectl patch installplan "$installPlan" -n rhacs --type merge --patch '{"spec":{"approved":true}}'
