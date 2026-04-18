FROM registry.access.redhat.com/ubi9/go-toolset:1.25.8@sha256:4b7a967ce9c283ecfb1e7a186f7b97d3f35a479bcc38d3c03a0707f3c36e974d AS build
USER root
ENV GOFLAGS="-mod=mod"

RUN mkdir /src
WORKDIR /src
COPY go.*  ./
RUN go mod download
COPY . ./

RUN make emailsender

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7@sha256:fe688da81a696387ca53a4c19231e99289591f990c904ef913c51b6e87d4e4df AS standard

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
