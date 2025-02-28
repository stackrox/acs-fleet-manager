FROM registry.access.redhat.com/ubi9/ubi-minimal:9.5-1738816775 AS build
ENV HOME=/opt/helm
COPY rhacs-terraform  ${HOME}/rhacs-terraform
WORKDIR ${HOME}

RUN curl -L --retry 10 --silent --show-error --fail -o /usr/local/bin/helm \
    "https://mirror.openshift.com/pub/openshift-v4/clients/helm/latest/helm-linux-amd64" && \
    chmod +x /usr/local/bin/helm && \
    helm version

RUN helm repo add external-secrets "https://charts.external-secrets.io/" && \
    helm dependencies build ${HOME}/rhacs-terraform

RUN microdnf install gzip tar && \
    curl -L --retry 10 --silent --show-error --fail -o /tmp/yq_linux_amd64.tar.gz \
        "https://github.com/mikefarah/yq/releases/download/v4.44.5/yq_linux_amd64.tar.gz" && \
        tar -xzf /tmp/yq_linux_amd64.tar.gz ./yq_linux_amd64 && \
        mv yq_linux_amd64 /usr/local/bin/yq && \
        chmod +x /usr/local/bin/yq && \
        rm /tmp/yq_linux_amd64.tar.gz

# Fix ignored securityContext.runAsUser set to null in values.yaml file.
# Manually dropping  securityContext.runAsUser value from the external secret subchart.
# This could be fixed with ose-helm-operator version bump.
# See: https://github.com/operator-framework/operator-sdk/issues/6635
RUN cd rhacs-terraform/charts && for filename in *.tgz; do tar -xf "$filename" && rm -f "$filename"; done && \
        yq -i 'del(.securityContext.runAsUser) | del(.webhook.securityContext.runAsUser) | del(.certController.securityContext.runAsUser)' external-secrets/values.yaml

ARG IMAGE_TAG=latest
RUN yq -i ".global.image.tag = strenv(IMAGE_TAG)" rhacs-terraform/values.yaml

# RH catalog see: https://catalog.redhat.com/software/containers/openshift4/ose-helm-operator
FROM registry.redhat.io/openshift4/ose-helm-rhel9-operator:v4.17.0-202502051104.p0.g61a705e.assembly.stream.el9

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
