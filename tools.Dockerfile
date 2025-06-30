FROM --platform=$BUILDPLATFORM registry.access.redhat.com/ubi8/go-toolset:1.23.9-2.1750813114 AS build
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

COPY . ./

ARG TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=cache,target=/go/.cache/ \
    make binary GOARCH=${TARGETARCH}

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.10-1295 AS standard

ENV KUBECTL_VERSION=v1.28.1

COPY --from=build /src/fleet-manager /src/fleetshard-sync /src/acsfleetctl /usr/local/bin/

RUN microdnf install tar gzip

# Install kubeval
RUN curl -LO https://github.com/instrumenta/kubeval/releases/download/v0.16.1/kubeval-linux-amd64.tar.gz
RUN curl -LO "https://github.com/instrumenta/kubeval/releases/download/v0.16.1/checksums.txt"
RUN cat checksums.txt | grep linux-amd64 | sha256sum --check
RUN tar -xf kubeval-linux-amd64.tar.gz

RUN mv kubeval /usr/bin/kubeval
RUN chmod +x /usr/bin/kubeval
RUN rm kubeval-linux-amd64.tar.gz

# Install kubectl
RUN curl -o /usr/bin/kubectl -LO "https://dl.k8s.io/release/$KUBECTL_VERSION/bin/linux/amd64/kubectl"
RUN chmod +x /usr/bin/kubectl
RUN curl -LO "https://dl.k8s.io/$KUBECTL_VERSION/bin/linux/amd64/kubectl.sha256"
RUN echo "$(cat kubectl.sha256)  /usr/bin/kubectl" | sha256sum --check

LABEL name="fleet-manager-tools" \
      vendor="Red Hat" \
      version="0.0.1" \
      summary="FleetManagerTools" \
      description="RHACS fleet-manager tools used for CI pipelines"
