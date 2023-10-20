FROM registry.ci.openshift.org/openshift/release:golang-1.21 AS build

RUN mkdir /rds_ca
ADD https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem /rds_ca/aws-rds-ca-global-bundle.pem

RUN mkdir /src
WORKDIR /src
COPY . ./

RUN make binary

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.8 as standard

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
