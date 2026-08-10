FROM --platform=$BUILDPLATFORM registry.access.redhat.com/ubi9/go-toolset:1.26.5@sha256:981cd8d53cc230b928db75a697586f915da5d3ed6731eddb246ce8865e0b1400 AS build
USER root
ENV GOFLAGS="-mod=mod"

RUN mkdir /src
WORKDIR /src
COPY go.*  ./
RUN go mod download
COPY . ./

ARG TARGETARCH

RUN make probe GOOS=linux GOARCH=${TARGETARCH}

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.8@sha256:463cae32c6f6f5594b11a5c22de275016bd8545ce58a6373388e8b24f13fc15c as standard

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
