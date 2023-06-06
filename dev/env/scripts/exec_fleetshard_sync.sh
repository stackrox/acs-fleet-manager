#!/usr/bin/env bash

set -euo pipefail

GITROOT="$(git rev-parse --show-toplevel)"
ENABLE_EXTERNAL_CONFIG="${ENABLE_EXTERNAL_CONFIG:-true}"

ARGS="${GITROOT}/fleetshard-sync"
if [[ "$#" -gt 0 ]]; then
    ARGS="$*"
fi

if [[ "$ENABLE_EXTERNAL_CONFIG" != "true" ]]; then
    ${ARGS}
    exit
fi

export AWS_AUTH_HELPER="${AWS_AUTH_HELPER:-aws-saml}"
# shellcheck source=scripts/lib/external_config.sh
source "${GITROOT}/scripts/lib/external_config.sh"
init_chamber

CLUSTER_NAME="cluster-acs-dev-dp-01"
ARGS="$ARGS"

#chamber exec fleetshard-sync -b secretsmanager -- sh -c "$ARGS"

helm_args=(
    "--set fleetshardSync.managedDB.enabled=${MANAGED_DB_ENABLED}"
)

if [[ "${MANAGED_DB_ENABLED}" -eq "true" ]]; then
    helm_args+=(
        "--set fleetshardSync.managedDB.subnetGroup=${MANAGED_DB_SECURITY_GROUP:-$(chamber read ${CLUSTER_NAME} MANAGED_DB_SECURITY_GROUP -q -b ssm)}"
        "--set fleetshardSync.managedDB.securityGroup=MANAGED_DB_SUBNET_GROUP=${MANAGED_DB_SUBNET_GROUP:-$(chamber read ${CLUSTER_NAME} MANAGED_DB_SUBNET_GROUP -q -b ssm)}"
        "--set fleetshardSync.managedDB.performanceInsights=${MANAGED_DB_PERFORMANCE_INSIGHTS}"
    )
fi

helm upgrade --install ${GITROOT}/dp-terraform/helm/rhacs-terraform \
    --namespace "$ACSMS_NAMESPACE" \
    --set fleetshardSync.clusterid="${CLUSTER_ID:-$(chamber read ${CLUSTER_NAME} ID -q -b ssm)}" \
    --set fleetshardSync.clusterName="${CLUSTER_NAME}" \
    --set fleetshardSync.createAuthProvider="false" \
    --set fleetshardSync.authType="" \
    --set fleetshardSync.managedDB.enabled="${MANAGED_DB_ENABLED}" \
    --set fleetshardSync.image="${FLEET_MANAGER_IMAGE}" \
    --set fleetshardSync.aws.roleARN="${FLEETSHARD_SYNC_AWS_ROLE_ARN:-$(chamber read fleetshard-sync AWS_ROLE_ARN -q -b ssm)}" \
    --set fleetshardSync.fleetManagerEndpoint="http://fleet-manager:8000" \
    --set fleetshardSync.redHatSSO.clientId="${RHSSO_SERVICE_ACCOUNT_CLIENT_ID}" \
    --set fleetshardSync.redHatSSO.clientSecret="${RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET}"

#    --set fleetshardSync.resources.limits.cpu="" \
#    --set fleetshardSync.resources.limits.memory="" \
#    --set fleetshardSync.resources.requests="" \
#    --set fleetshardSync.resources.requests.memory=""
