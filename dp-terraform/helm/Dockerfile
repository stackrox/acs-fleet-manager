# DEPRECATED: Use fleetshard-operator/Dockerfile instead.
# This file is used in Openshift CI only until it's changed in the openshift/release repo.
# Can't change both in one go.
FROM registry.access.redhat.com/ubi8/ubi-minimal:8.10-1295 AS build
ENV HOME=/opt/helm
COPY rhacs-terraform  ${HOME}/rhacs-terraform
WORKDIR ${HOME}

RUN microdnf install gzip tar && \
    curl -L --retry 10 --silent --show-error --fail -o /tmp/yq_linux_amd64.tar.gz \
        "https://github.com/mikefarah/yq/releases/download/v4.44.5/yq_linux_amd64.tar.gz" && \
        tar -xzf /tmp/yq_linux_amd64.tar.gz ./yq_linux_amd64 && \
        mv yq_linux_amd64 /usr/local/bin/yq && \
        chmod +x /usr/local/bin/yq && \
        rm /tmp/yq_linux_amd64.tar.gz

ARG IMAGE_TAG=latest
RUN yq -i ".global.image.tag = strenv(IMAGE_TAG)" rhacs-terraform/values.yaml

# RH catalog see: https://catalog.redhat.com/software/containers/openshift4/ose-helm-operator
FROM registry.redhat.io/openshift4/ose-helm-operator:v4.15.0-202503310336.p0.g52fc4b9.assembly.stream.el8

ENV HOME=/opt/helm
ENV ADDON_NAME=acs-fleetshard
COPY watches.yaml ${HOME}/watches.yaml
COPY --from=build ${HOME}/rhacs-terraform ${HOME}/rhacs-terraform
WORKDIR ${HOME}

LABEL name="acs-fleetshard-operator" \
    vendor="Red Hat, Inc." \
    version="0.0.1" \
    summary="ACSCS fleetshard Operator" \
    description="Red Hat Advanced Cluster Security Fleetshard Operator for Addon"
