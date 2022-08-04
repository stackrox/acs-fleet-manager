#!/bin/bash -ex
#
# This script runs E22 tests against stage environment on app-interface CI for ACS Fleet manager.
#
# FLEET_MANAGER_ENDPOINT - The base URL endpoint of the sage fleet manager instance.
#
# STATIC_OCM_TOKEN - The static token for the SSO.
#
# By default AWS provider on `us-east-1` region is used because stage Data Plane is configured that way.
#

echo "ACS fleet manager base url: $ACS_FLEET_MANAGER_ENDPOINT"

make \
  CLUSTER_ID="1smhq7nc0ncfv2jbjgf48q7e6qb943ou" \
  DP_CLOUD_PROVIDER="aws" \
  DP_REGION="us-east-1" \
  FLEET_MANAGER_ENDPOINT="${ACS_FLEET_MANAGER_ENDPOINT}" \
  AUTH_TYPE="STATIC_TOKEN" \
  STATIC_TOKEN="${STATIC_OCM_TOKEN}" \
  test/e2e
