FROM registry.access.redhat.com/ubi9/go-toolset:1.26.7@sha256:8cf89835994846ca0dffb9078e3a5638c57ec6175750f0af02fbe9c9942696d3 AS build
USER root
ENV GOFLAGS="-mod=mod"

RUN mkdir /src
WORKDIR /src
COPY go.*  ./
RUN go mod download
COPY . ./

RUN make emailsender

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.8@sha256:463cae32c6f6f5594b11a5c22de275016bd8545ce58a6373388e8b24f13fc15c AS standard

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
