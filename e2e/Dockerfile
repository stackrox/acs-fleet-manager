FROM registry.access.redhat.com/ubi8-minimal:8.6 AS builder

RUN microdnf install -y tar gzip make which git

# install go 1.18.8
RUN curl -O -J https://dl.google.com/go/go1.18.8.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.18.8.linux-amd64.tar.gz
RUN ln -s /usr/local/go/bin/go /usr/local/bin/go

WORKDIR /workspace

COPY . ./

RUN go mod download
RUN make probe

FROM registry.access.redhat.com/ubi8-minimal:8.6

COPY --from=builder /workspace/probe/bin /usr/local/bin/

ENTRYPOINT [ "/usr/local/bin/probe", "run" ]

LABEL name="e2e-acs-fleet-manager" \
      vendor="Red Hat" \
      version="0.0.1" \
      summary="E2eAcsFleetManager" \
      description="E2E ACS Fleet Manager"
