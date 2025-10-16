#!/usr/bin/env bash

# This script is to enable developers to work with local openshift builds
# It works well with OSD on AWS image flavor

# What it does
# - It sets up a BuildConfig and an ImageStream for fleet-manager and fleetshard-operator
# - It sets up the annotations on the deployments to use those images
# - It creates a build

# This assumes that make deploy/bootstrap and make deploy/dev have been run

CUR_BRANCH=$(git rev-parse --abbrev-ref HEAD)

# Set up the BuildConfig and ImageStream
oc apply -f - <<EOF
kind: ImageStream
apiVersion: image.openshift.io/v1
metadata:
    name: fleet-manager
    namespace: rhacs
spec:
    lookupPolicy:
        local: true
---
kind: ImageStream
apiVersion: image.openshift.io/v1
metadata:
    name: fleetshard-operator
    namespace: rhacs
spec:
    lookupPolicy:
        local: true
---
kind: BuildConfig
apiVersion: build.openshift.io/v1
metadata:
    name: fleet-manager
    namespace: rhacs
spec:
    successfulBuildsHistoryLimit: 5
    failedBuildsHistoryLimit: 5
    output:
        to:
            kind: ImageStreamTag
            name: 'fleet-manager:latest'
    strategy:
        type: Docker
        dockerStrategy:
            dockerfilePath: Dockerfile
    source:
        type: Git
        git:
            uri: 'https://github.com/stackrox/acs-fleet-manager'
            ref: ${CUR_BRANCH}
        contextDir: /
    runPolicy: Serial
---
kind: BuildConfig
apiVersion: build.openshift.io/v1
metadata:
    name: fleetshard-operator
    namespace: rhacs
spec:
    successfulBuildsHistoryLimit: 5
    failedBuildsHistoryLimit: 5
    output:
        to:
            kind: ImageStreamTag
            name: 'fleetshard-operator:latest'
    strategy:
        type: Docker
        dockerStrategy:
            dockerfilePath: Dockerfile
    source:
        type: Git
        git:
            uri: 'https://github.com/stackrox/acs-fleet-manager'
            ref: ${CUR_BRANCH}
        contextDir: /fleetshard-operator
    runPolicy: Serial
EOF

oc annotate deployment -n rhacs acs-fleetshard-operator image.openshift.io/triggers='[{"from":{"kind":"ImageStreamTag","name":"fleetshard-operator:latest","namespace":"rhacs"},"fieldPath":"spec.template.spec.containers[?(@.name==\"manager\")].image"}]' --overwrite
oc annotate deployment -n rhacs fleet-manager image.openshift.io/triggers='[{"from":{"kind":"ImageStreamTag","name":"fleet-manager:latest","namespace":"rhacs"},"fieldPath":"spec.template.spec.containers[?(@.name==\"service\")].image"},{"from":{"kind":"ImageStreamTag","name":"fleet-manager:latest","namespace":"rhacs"},"fieldPath":"spec.template.spec.containers[?(@.name==\"migration\")].image"}]' --overwrite
oc delete deployment -n rhacs acs-fleetshard-operator
oc annotate deployment -n rhacs fleetshard-sync image.openshift.io/triggers='[{"from":{"kind":"ImageStreamTag","name":"fleet-manager:latest","namespace":"rhacs"},"fieldPath":"spec.template.spec.containers[?(@.name==\"fleetshard-sync\")].image"}]' --overwrite

oc start-build -n rhacs fleet-manager
oc start-build -n rhacs fleetshard-operator
