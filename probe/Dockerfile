FROM --platform=$BUILDPLATFORM registry.access.redhat.com/ubi9/go-toolset:1.25.8@sha256:77bfb0f283eaa3215909342c3dda940605eff5b9f72d6dc18fad1d154d172d55 AS build
USER root
ENV GOFLAGS="-mod=mod"

RUN mkdir /src
WORKDIR /src
COPY go.*  ./
RUN go mod download
COPY . ./

ARG TARGETARCH

RUN make probe GOOS=linux GOARCH=${TARGETARCH}

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7@sha256:7d4e47500f28ac3a2bff06c25eff9127ff21048538ae03ce240d57cf756acd00 as standard

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
