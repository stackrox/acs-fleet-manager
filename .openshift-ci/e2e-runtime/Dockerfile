FROM quay.io/centos/centos:stream9

# Workaround for https://bugzilla.redhat.com/show_bug.cgi?id=2184640
RUN rpm --import /etc/pki/rpm-gpg/RPM-GPG-KEY-centosofficial

RUN dnf update -y --disablerepo=\* --enablerepo=baseos,appstream && dnf -y install procps make which git gettext jq gcc && dnf clean all && rm -rf /var/cache/dnf

COPY --from=registry.ci.openshift.org/openshift/release:golang-1.23 /usr/lib/golang /usr/lib/golang
COPY --from=quay.io/openshift/origin-cli:4.20 /usr/bin/oc /usr/bin

ENV GOPATH=/go
ENV GOROOT=/usr/lib/golang
ENV PATH="${GOROOT}/bin:${PATH}"

RUN ln -s /usr/bin/oc /usr/bin/kubectl

ARG YQ_VERSION=4.27.5
RUN curl -L --retry 10 --silent --show-error --fail -o /tmp/yq_linux_amd64.tar.gz \
    "https://github.com/mikefarah/yq/releases/download/v${YQ_VERSION}/yq_linux_amd64.tar.gz" && \
    tar -xzf /tmp/yq_linux_amd64.tar.gz ./yq_linux_amd64 && \
    mv yq_linux_amd64 /usr/local/bin/yq && \
    chmod +x /usr/local/bin/yq && \
    rm /tmp/yq_linux_amd64.tar.gz

ARG STERN_VERSION="1.22.0"
RUN curl -L --retry 10 --silent --show-error --fail -o "/tmp/stern_linux_amd64.tar.gz" \
    "https://github.com/stern/stern/releases/download/v${STERN_VERSION}/stern_${STERN_VERSION}_linux_amd64.tar.gz" && \
    tar -xf /tmp/stern_linux_amd64.tar.gz stern && \
    mv stern /usr/local/bin/stern && \
    chmod +x /usr/local/bin/stern

ARG OCM_VERSION=0.1.64
RUN curl -L --retry 10 --silent --show-error --fail -o "/usr/local/bin/ocm" \
    "https://github.com/openshift-online/ocm-cli/releases/download/v${OCM_VERSION}/ocm-linux-amd64" && \
    chmod +x /usr/local/bin/ocm

RUN mkdir /src $GOPATH
WORKDIR /src

COPY . .
RUN chmod 775 -R /src && chmod 775 -R $GOPATH && git config --system --add safe.directory /src

CMD ./.openshift-ci/tests/e2e.sh
