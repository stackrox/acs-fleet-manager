FROM registry.access.redhat.com/ubi9/go-toolset:1.25.7@sha256:3cdf0d1e8d8601c89fe3070d7aa63a54d1aaf049ba55fba314f6358b1d95a2f9 AS build
USER root
ENV GOFLAGS="-mod=mod"

RUN mkdir /src
WORKDIR /src
COPY go.*  ./
RUN go mod download
COPY . ./

RUN make emailsender

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7@sha256:83006d535923fcf1345067873524a3980316f51794f01d8655be55d6e9387183 AS standard

RUN microdnf install shadow-utils

RUN useradd -u 1001 unprivilegeduser
ADD https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem /rds_ca/aws-rds-ca-global-bundle.pem
RUN chmod a+rw /rds_ca/aws-rds-ca-global-bundle.pem
# Switch to non-root user
USER unprivilegeduser

COPY --from=build /src/emailsender/bin /acscs/
EXPOSE 8080
ENTRYPOINT ["/acscs/emailsender"]
LABEL name="ACSCS email sender" \
	vendor="Red Hat, Inc." \
	version="0.0.1" \
	summary="Email sender service for ACSCS" \
	description="Email sender service for the Red Hat Advanced Cluster Security Cloud Services"
