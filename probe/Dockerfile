FROM --platform=$BUILDPLATFORM registry.access.redhat.com/ubi9/go-toolset:1.26.7@sha256:5e68f09a652ac6627a83c57655e42e24575efb278b54336039c9308607fc6b21 AS build
USER root
ENV GOFLAGS="-mod=mod"

RUN mkdir /src
WORKDIR /src
COPY go.*  ./
RUN go mod download
COPY . ./

ARG TARGETARCH

RUN make probe GOOS=linux GOARCH=${TARGETARCH}

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.8@sha256:7fbeae18dc9476399f565e68255f602a3374ea8614ba3d14843565131a13ff93 as standard

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
