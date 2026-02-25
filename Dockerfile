FROM --platform=$BUILDPLATFORM registry.access.redhat.com/ubi9/go-toolset:1.25.7@sha256:799cc027d5ad58cdc156b65286eb6389993ec14c496cf748c09834b7251e78dc AS build

USER root
RUN mkdir /src /rds_ca
ADD https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem /rds_ca/aws-rds-ca-global-bundle.pem
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
    make fleet-manager fleetshard-sync GOOS=linux GOARCH=${TARGETARCH}

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7@sha256:c7d44146f826037f6873d99da479299b889473492d3c1ab8af86f08af04ec8a0 AS standard

RUN microdnf install shadow-utils

RUN useradd -u 1001 unprivilegeduser
# Switch to non-root user
USER unprivilegeduser

COPY --chown=unprivilegeduser --from=build /src/fleet-manager /src/fleetshard-sync /usr/local/bin/
COPY --chown=unprivilegeduser --from=build /rds_ca /usr/local/share/ca-certificates

EXPOSE 8000
WORKDIR /
ENTRYPOINT ["/usr/local/bin/fleet-manager", "serve"]

LABEL name="fleet-manager" \
    vendor="Red Hat, Inc." \
    version="0.0.1" \
    summary="FleetManager" \
    description="Red Hat Advanced Cluster Security Fleet Manager"
