FROM --platform=$BUILDPLATFORM registry.access.redhat.com/ubi9/go-toolset:1.25.3@sha256:e8938564f866174a6d79e55dfe577c2ed184b1f53e91d782173fb69b07ce69ef AS build
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
    make fleetshard-operator GOOS=linux GOARCH=${TARGETARCH}

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7@sha256:90bd85dcd061d1ad6dbda70a867c41958c04a86462d05c631f8205e8870f28f8

RUN microdnf install shadow-utils

RUN useradd -u 1001 unprivilegeduser
# Switch to non-root user
USER unprivilegeduser

COPY --chown=unprivilegeduser --from=build /src/fleetshard-operator/bin /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/fleetshard-operator"]

LABEL name="acs-fleetshard-operator" \
    vendor="Red Hat, Inc." \
    version="0.0.1" \
    summary="ACSCS fleetshard Operator" \
    description="Red Hat Advanced Cluster Security Fleetshard Operator for Addon"
