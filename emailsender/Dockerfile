FROM registry.access.redhat.com/ubi9/go-toolset:1.26.7@sha256:5e68f09a652ac6627a83c57655e42e24575efb278b54336039c9308607fc6b21 AS build
USER root
ENV GOFLAGS="-mod=mod"

RUN mkdir /src
WORKDIR /src
COPY go.*  ./
RUN go mod download
COPY . ./

RUN make emailsender

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.8@sha256:7fbeae18dc9476399f565e68255f602a3374ea8614ba3d14843565131a13ff93 AS standard

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
