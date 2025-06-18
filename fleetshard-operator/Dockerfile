FROM registry.access.redhat.com/ubi8/ubi-minimal:8.10-1295 AS build-helm
ENV HOME=/opt/helm
COPY ../dp-terraform/helm/rhacs-terraform  ${HOME}/rhacs-terraform
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

FROM --platform=$BUILDPLATFORM registry.access.redhat.com/ubi8/go-toolset:1.23.6-4 AS build-go
USER root

RUN mkdir /src
WORKDIR /src

RUN go env -w GOCACHE=/go/.cache; \
    go env -w GOMODCACHE=/go/pkg/mod

# Use the docker build cache to avoid calling 'go mod download' if go.mod/go.sum have not been changed.
# Otherwise, use cache mount to cache dependencies between builds.
# mount=type=bind is intentionally not used to ensure compatibility between docker and podman.
# See:
#  - https://docs.docker.com/build/cache/
#  - https://docs.docker.com/build/guide/mounts/
#  - https://github.com/containers/podman/issues/15423
COPY go.*  ./
RUN --mount=type=cache,target=/go/pkg/mod/ \
      go mod download
COPY .. ./

ARG TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=cache,target=/go/.cache/ \
    make fleetshard-operator GOOS=linux GOARCH=${TARGETARCH}

# RH catalog see: https://catalog.redhat.com/software/containers/openshift4/ose-helm-operator
FROM registry.redhat.io/openshift4/ose-helm-operator:v4.15.0-202503310336.p0.g52fc4b9.assembly.stream.el8

ENV HOME=/opt/helm
ENV ADDON_NAME=acs-fleetshard
COPY ../dp-terraform/helm/watches.yaml ${HOME}/watches.yaml
COPY --from=build-helm ${HOME}/rhacs-terraform ${HOME}/rhacs-terraform
COPY --from=build-go /src/fleetshard-operator/bin /usr/local/bin/
WORKDIR ${HOME}

LABEL name="acs-fleetshard-operator" \
    vendor="Red Hat, Inc." \
    version="0.0.1" \
    summary="ACSCS fleetshard Operator" \
    description="Red Hat Advanced Cluster Security Fleetshard Operator for Addon"
