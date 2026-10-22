FROM registry.access.redhat.com/ubi9/go-toolset:1.26.10@sha256:d4d25f236c2b3cd27eb407e3b3478bef0b75d644ee6d11772217f537be926b11 AS build
USER root
ENV GOFLAGS="-mod=mod"

RUN mkdir /src
WORKDIR /src
COPY go.*  ./
RUN go mod download
COPY . ./

RUN make emailsender

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.8@sha256:7b8e25a1b56ca4d00219198f3b5b51a3e1693a5c4f5369c5e190d7d6cb3f980e AS standard

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
