FROM --platform=$BUILDPLATFORM registry.access.redhat.com/ubi9/go-toolset:1.25.7@sha256:3cdf0d1e8d8601c89fe3070d7aa63a54d1aaf049ba55fba314f6358b1d95a2f9 AS build
USER root
ENV GOFLAGS="-mod=mod"

RUN mkdir /src
WORKDIR /src
COPY go.*  ./
RUN go mod download
COPY . ./

ARG TARGETARCH

RUN make probe GOOS=linux GOARCH=${TARGETARCH}

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7@sha256:69f5c9886ecb19b23e88275a5cd904c47dd982dfa370fbbd0c356d7b1047ef68 as standard

RUN microdnf install shadow-utils

RUN useradd -u 1001 unprivilegeduser
# Switch to non-root user
USER unprivilegeduser

COPY --from=build /src/probe/bin /stackrox/
EXPOSE 7070
ENTRYPOINT ["/stackrox/probe"]
CMD ["start"]
LABEL name="probe" \
	vendor="Red Hat, Inc." \
	version="0.0.1" \
	summary="Blackbox monitoring probe for ACS Fleet Manager" \
	description="Blackbox monitoring probe for the Red Hat Advanced Cluster Security Fleet Manager"
