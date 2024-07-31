FROM --platform=$BUILDPLATFORM registry.access.redhat.com/ubi8/ubi-minimal:8.10 AS build
ENV HOME=/opt/helm
COPY rhacs-terraform  ${HOME}/rhacs-terraform
WORKDIR ${HOME}

ARG TARGETARCH

RUN curl -L --retry 10 --silent --show-error --fail -o /usr/local/bin/helm \
    "https://mirror.openshift.com/pub/openshift-v4/clients/helm/latest/helm-linux-${TARGETARCH}" && \
    chmod +x /usr/local/bin/helm && \
    helm version

RUN helm repo add external-secrets "https://charts.external-secrets.io/" && \
    helm dependencies build ${HOME}/rhacs-terraform

# Workaround for deleting securityContext.runAsUser from the dependent chart
# see: https://github.com/operator-framework/operator-sdk/issues/6635
RUN microdnf install gzip tar && \
    curl -L --retry 10 --silent --show-error --fail -o /tmp/yq.tar.gz \
        "https://github.com/mikefarah/yq/releases/download/v4.44.2/yq_linux_${TARGETARCH}.tar.gz" && \
        tar -xzf /tmp/yq.tar.gz ./yq_linux_${TARGETARCH} && \
        mv yq_linux_${TARGETARCH} /usr/local/bin/yq && \
        chmod +x /usr/local/bin/yq && \
        rm /tmp/yq.tar.gz && \
    cd rhacs-terraform/charts && for filename in *.tgz; do tar -xf "$filename" && rm -f "$filename"; done && \
    yq -i 'del(.securityContext.runAsUser)' external-secrets/values.yaml

ARG FLEETSHARD_SYNC_IMAGE_TAG=main
RUN yq -i ".fleetshardSync.image.tag = strenv(FLEETSHARD_SYNC_IMAGE_TAG)" rhacs-terraform/values.yaml
ARG EMAILSENDER_IMAGE_TAG=main
RUN yq -i ".emailsender.image.tag = strenv(EMAILSENDER_IMAGE_TAG)" rhacs-terraform/values.yaml

FROM quay.io/operator-framework/helm-operator:v1.33.0

ENV HOME=/opt/helm
ENV ADDON_NAME=acs-fleetshard
COPY watches.yaml ${HOME}/watches.yaml
COPY --from=build ${HOME}/rhacs-terraform ${HOME}/rhacs-terraform
WORKDIR ${HOME}
