#!/bin/bash -ex
#
# This script runs the probe based E2E tests against stage environment on app-interface CI for ACS Fleet manager.
# You can find e2e app-interface CI configuration here:
# https://gitlab.cee.redhat.com/service/app-interface/-/blob/master/data/services/acs-fleet-manager/cicd/jobs.yaml
#
# The following environment variables should be provided for running e2e tests:
#
# OCM_USERNAME - the OCM user name, It will owner of provisioned ACS instance.
#
# OCM_TOKEN - The static OCM token for the SSO.
#
# FLEET_MANAGER_ENDPOINT - The base URL endpoint of the sage fleet manager instance.
#
# Env variables maps from AppSRE Vault in the CI
#

make \
    OCM_USERNAME="${OCM_USERNAME}" \
    OCM_TOKEN="${STATIC_OCM_TOKEN}" \
    FLEET_MANAGER_ENDPOINT="${ACS_FLEET_MANAGER_ENDPOINT}" \
  test/e2e/probe/run
