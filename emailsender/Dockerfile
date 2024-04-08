FROM registry.access.redhat.com/ubi8/go-toolset:1.20 AS build
USER root
ENV GOFLAGS="-mod=mod"

RUN mkdir /src
WORKDIR /src
RUN CGO_ENABLED=0 go install -ldflags "-s -w -extldflags '-static'" github.com/go-delve/delve/cmd/dlv@v1.20.1
COPY go.*  ./
RUN go mod download
COPY . ./

RUN make email-sender

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.9 as standard

RUN microdnf install shadow-utils

RUN useradd -u 1001 unprivilegeduser
# Switch to non-root user
USER unprivilegeduser

COPY --from=build /src/emailsender/bin /acscs/
EXPOSE 8000
ENTRYPOINT ["/acscs/emailsender"]
CMD ["start"]
LABEL name="ACSCS email sender" \
	vendor="Red Hat" \
	version="0.0.1" \
	summary="Email sender service for ACSCS" \
	description="Email sender service for the Red Hat Advanced Cluster Security Cloud Services"
