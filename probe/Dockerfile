FROM --platform=$BUILDPLATFORM registry.access.redhat.com/ubi9/go-toolset:1.26.5@sha256:e1ff4f548dfca084c2318d11dccd7d30a75c81a3a4fd8abcebc42ea42dd683e8 AS build
USER root
ENV GOFLAGS="-mod=mod"

RUN mkdir /src
WORKDIR /src
COPY go.*  ./
RUN go mod download
COPY . ./

ARG TARGETARCH

RUN make probe GOOS=linux GOARCH=${TARGETARCH}

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.8@sha256:1bc3c5c15720506a0cf48adfdf8b623dfe704377e007d7bbae8d14876392ca6a as standard

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
