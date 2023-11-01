#!/bin/bash -e
#
# Copyright (c) 2018 Red Hat, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

# =====================================================================================================================
# This script builds and pushes the ACS Fleet Manager service docker image on AppSRE JenkinsCI.
# You can find CI configuration in app-interface repository:
# https://gitlab.cee.redhat.com/service/app-interface/-/blob/master/data/services/acs-fleet-manager/cicd/jobs.yaml
# In order to work, it needs the following variables defined in the CI/CD configuration of the project:
#
# QUAY_USER - The name of the robot account used to push images to
# 'quay.io', for example 'openshift-unified-hybrid-cloud+jenkins'.
#
# QUAY_TOKEN - The token of the robot account used to push images to
# 'quay.io'.
#
# The machines that run this script need to have access to internet, so that
# the built images can be pushed to quay.io.
# =====================================================================================================================

# Set the quay organization to default value if it is not passed via env
QUAY_ORG=${QUAY_ORG:-app-sre}

source ./scripts/build_setup.sh

# Push the image:
echo "Quay.io user and token is set, will push images to $QUAY_ORG"
make \
  DOCKER_CONFIG="${DOCKER_CONFIG}" \
  QUAY_USER="${QUAY_USER}" \
  QUAY_TOKEN="${QUAY_TOKEN}" \
  external_image_registry="quay.io" \
  internal_image_registry="quay.io" \
  image_repository="${QUAY_ORG}/acs-fleet-manager" \
  docker/login/fleet-manager \
  image/push/fleet-manager

make \
  DOCKER_CONFIG="${DOCKER_CONFIG}" \
  QUAY_USER="${QUAY_USER}" \
  QUAY_TOKEN="${QUAY_TOKEN}" \
  TAG="${BRANCH}" \
  external_image_registry="quay.io" \
  internal_image_registry="quay.io" \
  image_repository="${QUAY_ORG}/acs-fleet-manager" \
  docker/login/fleet-manager \
  image/push/fleet-manager

make \
  DOCKER_CONFIG="${DOCKER_CONFIG}" \
  QUAY_USER="${QUAY_USER}" \
  QUAY_TOKEN="${QUAY_TOKEN}" \
  TAG="${VERSION}" \
  external_image_registry="quay.io" \
  internal_image_registry="quay.io" \
  image_repository="${QUAY_ORG}/acs-fleetshard-operator" \
  docker/login/fleet-manager \
  image/push/fleetshard-operator
