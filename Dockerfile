FROM registry.ci.openshift.org/openshift/release:golang-1.20 AS build

ENV GOFLAGS="-mod=mod"

RUN mkdir /rds_ca
ADD https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem /rds_ca/aws-rds-ca-global-bundle.pem

RUN mkdir /src
WORKDIR /src
RUN CGO_ENABLED=0 go install -ldflags "-s -w -extldflags '-static'" github.com/go-delve/delve/cmd/dlv@latest
COPY go.*  ./
RUN go mod download
COPY . ./

FROM build as build-debug
RUN GOARGS="-gcflags 'all=-N -l'" make binary

FROM build as build-standard
RUN make binary

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.8 as debug
COPY --from=build-debug /go/bin/dlv /src/fleet-manager /src/fleetshard-sync /usr/local/bin/
COPY --from=build-debug /src /src
COPY --from=build /rds_ca /usr/local/share/ca-certificates
EXPOSE 8000
WORKDIR /
ENTRYPOINT [ "/usr/local/bin/dlv" , "--listen=:40000", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "/usr/local/bin/fleet-manager", "serve"]

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.8 as standard

RUN microdnf install shadow-utils

RUN useradd -u 1001 unprivilegeduser
# Switch to non-root user
USER unprivilegeduser

COPY --chown=unprivilegeduser --from=build-standard /src/fleet-manager /src/fleetshard-sync /usr/local/bin/
COPY --chown=unprivilegeduser --from=build /rds_ca /usr/local/share/ca-certificates
EXPOSE 8000
WORKDIR /
ENTRYPOINT ["/usr/local/bin/fleet-manager", "serve"]

LABEL name="fleet-manager" \
    vendor="Red Hat" \
    version="0.0.1" \
    summary="FleetManager" \
    description="Red Hat Advanced Cluster Security Fleet Manager"
