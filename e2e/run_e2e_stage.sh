#!/bin/bash -ex
#
# This script runs E22 tests against stage environment on app-interface CI for ACS Fleet manager.
#
# FLEET_MANAGER_ENDPOINT - The base URL endpoint of the sage fleet manager instance.
#
# OCM_TOKEN - The static token for the SSO.
#
# By default AWS provider on `us-east-1` region is used because stage Data Plane is configured that way.
#

make \
  DP_CLOUD_PROVIDER="aws" \
  DP_REGION="us-east-1" \
  FLEET_MANAGER_ENDPOINT="${ACS_FLEET_MANAGER_ENDPOINT}" \
  OCM_TOKEN="${OCM_TOKEN}" \
  test/e2e/stage
