FROM registry.access.redhat.com/ubi8/s2i-base:1-388 AS build

ARG GO_VERSION=1.18.8
RUN curl -L --retry 10 --silent --show-error --fail -o /tmp/go.linux-amd64.tar.gz \
	"https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" && \
	tar -C /usr/local -xzf /tmp/go.linux-amd64.tar.gz && \
	rm -f /tmp/go.linux-amd64.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"

ARG GOPATH=/go
ENV GOPATH=${GOPATH}

ARG GOCACHE=/go/.cache
ENV GOCACHE=${GOCACHE}

ARG GOROOT=/usr/local/go
ENV GOROOT=${GOROOT}

ARG GOFLAGS=-mod=mod
ENV GOFLAGS=${GOFLAGS}

RUN mkdir /src
WORKDIR /src
RUN CGO_ENABLED=0 go install -ldflags "-s -w -extldflags '-static'" github.com/go-delve/delve/cmd/dlv@latest
COPY go.*  ./
RUN go mod download
COPY . ./

FROM build as build-debug
RUN GOARGS="-gcflags 'all=-N -l'" make probe

FROM build as build-standard
RUN make probe

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.6 as debug
COPY --from=build-debug /go/bin/dlv /src/probe/bin /stackrox/
COPY --from=build-debug /src /src
EXPOSE 7070
ENTRYPOINT [ "/stackrox/dlv" , "--listen=:40000", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "/stackrox/probe"]
CMD ["start"]

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.6 as standard
COPY --from=build-standard /src/probe/bin /stackrox/
EXPOSE 7070
ENTRYPOINT ["/stackrox/probe"]
CMD ["start"]
LABEL name="probe" \
	vendor="Red Hat" \
	version="0.0.1" \
	summary="Blackbox monitoring probe for ACS Fleet Manager" \
	description="Blackbox monitoring probe for the Red Hat Advanced Cluster Security Fleet Manager"
