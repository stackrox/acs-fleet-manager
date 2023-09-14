#!/bin/bash -e

# Set the directory for docker configuration:
export DOCKER_CONFIG="${PWD}/.docker"

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

export BRANCH="main"
if [[ -n "$GITHUB_REF" ]]; then
  BRANCH="$(echo "$GITHUB_REF" | awk -F/ '{print $NF}')"
  echo "GITHUB_REF is defined. Set image tag to $BRANCH."
elif [[ -n "$GIT_BRANCH" ]]; then
  BRANCH="$(echo "$GIT_BRANCH" | awk -F/ '{print $NF}')"
  echo "GIT_BRANCH is defined. Set image tag to $BRANCH."
else
  echo "No git branch env var found. Set image tag to $BRANCH."
fi
