#!/bin/bash -ex
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

# This script builds and deploys the Dinosaur Service Fleet Manager. In order to
# work, it needs the following variables defined in the CI/CD configuration of
# the project:
#
# QUAY_USER - The name of the robot account used to push images to
# 'quay.io', for example 'openshift-unified-hybrid-cloud+jenkins'.
#
# QUAY_TOKEN - The token of the robot account used to push images to
# 'quay.io'.
#
# The machines that run this script need to have access to internet, so that
# the built images can be pushed to quay.io.

# The version should be a 7-char hash from git. This is what the deployment process in app-interface expects.
VERSION=`git rev-parse --short=7 HEAD`

# Set image repository to default value if it is not passed via env
IMAGE_REPOSITORY="${QUAY_IMAGE_REPOSITORY:-app-sre/acs-fleet-manager}"

# Set the directory for docker configuration:
DOCKER_CONFIG="${PWD}/.docker"

# Set the Go path:
export GOPATH="${PWD}/.gopath"
# TODO(porridge): clean up this quick hack needed to get build working on jenkins
jenkins_go117="/opt/go/1.17.8"
export PATH="${jenkins_go117}/bin:${PATH}:${GOPATH}/bin"
LINK="${GOPATH}/src/github.com/stackrox/acs-fleet-manager"

# print go version
go version

mkdir -p "$(dirname "${LINK}")"
ln -sf "${PWD}" "${LINK}"
cd "${LINK}"

# Log in to the image registry:
if [ -z "${QUAY_USER}" ]; then
  echo "The quay.io push user name hasn't been provided."
  echo "Make sure to set the QUAY_USER environment variable."
  exit 1
fi
if [ -z "${QUAY_TOKEN}" ]; then
  echo "The quay.io push token hasn't been provided."
  echo "Make sure to set the QUAY_TOKEN environment variable."
  exit 1
fi

# Set up the docker config directory
mkdir -p "${DOCKER_CONFIG}"

BRANCH="main"
if [[ ! -z "$GITHUB_REF" ]]; then
  BRANCH="$(echo $GITHUB_REF | awk -F/ '{print $NF}')"
  echo "GITHUB_REF is defined. Set image tag to $BRANCH."
elif [[ ! -z "$GIT_BRANCH" ]]; then
  BRANCH="$(echo $GIT_BRANCH | awk -F/ '{print $NF}')"
  echo "GIT_BRANCH is defined. Set image tag to $BRANCH."
else
  echo "No git branch env var found. Set image tag to $BRANCH."
fi

# Push the image:
echo "Quay.io user and token is set, will push images to $IMAGE_REPOSITORY"
make \
  DOCKER_CONFIG="${DOCKER_CONFIG}" \
  QUAY_USER="${QUAY_USER}" \
  QUAY_TOKEN="${QUAY_TOKEN}" \
  TAG="${VERSION}" \
  external_image_registry="quay.io" \
  internal_image_registry="quay.io" \
  image_repository="${IMAGE_REPOSITORY}" \
  docker/login \
  image/push

make \
  DOCKER_CONFIG="${DOCKER_CONFIG}" \
  QUAY_USER="${QUAY_USER}" \
  QUAY_TOKEN="${QUAY_TOKEN}" \
  TAG="${BRANCH}" \
  external_image_registry="quay.io" \
  internal_image_registry="quay.io" \
  image_repository="${IMAGE_REPOSITORY}" \
  docker/login \
  image/push
