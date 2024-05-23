FROM --platform=$BUILDPLATFORM registry.access.redhat.com/ubi8/go-toolset:1.21 AS build

USER root
RUN mkdir /src /rds_ca
ADD https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem /rds_ca/aws-rds-ca-global-bundle.pem
WORKDIR /src

RUN go env -w GOCACHE=/go/.cache; \
    go env -w GOMODCACHE=/go/pkg/mod

RUN --mount=type=cache,target=/go/pkg/mod/ \
     --mount=type=bind,source=go.sum,target=go.sum \
     --mount=type=bind,source=go.mod,target=go.mod \
      go mod download -x

COPY . ./

ARG TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=cache,target=/go/.cache/ \
    make fleet-manager fleetshard-sync GOOS=linux GOARCH=${TARGETARCH}

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.9 as standard

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
    vendor="Red Hat" \
    version="0.0.1" \
    summary="FleetManager" \
    description="Red Hat Advanced Cluster Security Fleet Manager"
