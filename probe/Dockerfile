FROM registry.ci.openshift.org/openshift/release:golang-1.20 AS build

ENV GOFLAGS="-mod=mod"

RUN mkdir /src
WORKDIR /src
RUN CGO_ENABLED=0 go install -ldflags "-s -w -extldflags '-static'" github.com/go-delve/delve/cmd/dlv@v1.20.1
COPY go.*  ./
RUN go mod download
COPY . ./

FROM build as build-debug
RUN GOARGS="-gcflags 'all=-N -l'" make probe

FROM build as build-standard
RUN make probe

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.8 as debug
COPY --from=build-debug /go/bin/dlv /src/probe/bin /stackrox/
COPY --from=build-debug /src /src
EXPOSE 7070
ENTRYPOINT [ "/stackrox/dlv" , "--listen=:40000", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "/stackrox/probe"]
CMD ["start"]

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.8 as standard

RUN microdnf install shadow-utils

RUN useradd -u 1001 unprivilegeduser
# Switch to non-root user
USER unprivilegeduser

COPY --from=build-standard /src/probe/bin /stackrox/
EXPOSE 7070
ENTRYPOINT ["/stackrox/probe"]
CMD ["start"]
LABEL name="probe" \
	vendor="Red Hat" \
	version="0.0.1" \
	summary="Blackbox monitoring probe for ACS Fleet Manager" \
	description="Blackbox monitoring probe for the Red Hat Advanced Cluster Security Fleet Manager"
