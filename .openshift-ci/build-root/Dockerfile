# Changes to this file are not validated automatically by CI. That is because
# the CI as defined in openshift/release runs against HEAD and uses the version
# of this file found there.

# In order to validate a change to this file i.e. a new version of the test environment:
# - make the change on a stackrox/stackrox PR (do not use / in the branch name as it is not supported in openshift/release)
# - open a PR in openshift/release (this is just for test. mark the PR with `/hold` and `/uncc` autoassigned reviewers to reduce noise)
# - duplicate the main branch CI workflow to a workflow that tests the stackrox/stackrox PR branch
# - run openshift/release automation to generate the prow config
# - `make update` and commit the results
# - run `/test pj-rehearse-max` on the openshift/release PR to validate the change

FROM quay.io/centos/centos:stream9

RUN rm -f /etc/yum.repos.d/* && { \
    echo "[baseos]"; \
    echo "name=CentOS Stream \$releasever - BaseOS"; \
    echo "baseurl=http://mirror.stream.centos.org/\$releasever-stream/BaseOS/\$basearch/os/"; \
    echo "gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-centosofficial"; \
    echo "gpgcheck=1"; \
    echo "repo_gpgcheck=0"; \
    echo "metadata_expire=6h"; \
    echo "countme=1"; \
    echo "enabled=1"; \
    echo; \
    echo "[appstream]"; \
    echo "name=CentOS Stream $releasever - AppStream"; \
    echo "baseurl=http://mirror.stream.centos.org/\$releasever-stream/AppStream/\$basearch/os/"; \
    echo "gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-centosofficial"; \
    echo "gpgcheck=1"; \
    echo "repo_gpgcheck=0"; \
    echo "metadata_expire=6h"; \
    echo "countme=1"; \
    echo "enabled=1"; \
    } > "/etc/yum.repos.d/centos.repo"

RUN dnf update -y && dnf -y install make which git gettext jq gcc

RUN curl -L --retry 10 --silent --show-error --fail -o /tmp/go1.17.12.linux-amd64.tar.gz \
    https://go.dev/dl/go1.17.12.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf /tmp/go1.17.12.linux-amd64.tar.gz && \
    rm -f /tmp/go1.17.12.linux-amd64.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"

RUN curl -L --retry 10 --silent --show-error --fail -o "/usr/local/bin/yq" \
    https://github.com/mikefarah/yq/releases/download/v4.2.0/yq_linux_amd64 && \
    chmod +x /usr/local/bin/yq

RUN curl -L --retry 10 --silent --show-error --fail -O https://nodejs.org/dist/v16.15.1/node-v16.15.1-linux-x64.tar.gz && \
    mkdir -p /usr/local/lib/nodejs && \
    tar -xzf node-v16.15.1-linux-x64.tar.gz -C /usr/local/lib/nodejs && \
    rm node-v16.15.1-linux-x64.tar.gz
ENV PATH="/usr/local/lib/nodejs/node-v16.15.1-linux-x64/bin:${PATH}"

RUN curl -L --retry 10 --silent --show-error --fail -o "/usr/local/bin/stern" \
    https://github.com/wercker/stern/releases/download/1.11.0/stern_linux_amd64 && \
    chmod +x /usr/local/bin/stern

RUN curl -L --retry 10 --silent --show-error --fail -O https://github.com/gotestyourself/gotestsum/releases/download/v1.8.1/gotestsum_1.8.1_linux_amd64.tar.gz && \
    tar -xzvf gotestsum_1.8.1_linux_amd64.tar.gz gotestsum && \
    mv gotestsum /usr/local/bin && \
    chmod +x /usr/local/bin/gotestsum && \
    rm gotestsum_1.8.1_linux_amd64.tar.gz

RUN mkdir -p /stackrox/crds && \
    curl -L --retry 10 --silent --show-error --fail -o /stackrox/crds/platform.stackrox.io_centrals.yaml \
    https://raw.githubusercontent.com/stackrox/stackrox/release/3.70.x/operator/config/crd/bases/platform.stackrox.io_centrals.yaml && \
    curl -L --retry 10 --silent --show-error --fail -o /stackrox/crds/platform.stackrox.io_securedclusters.yaml \
    https://raw.githubusercontent.com/stackrox/stackrox/release/3.70.x/operator/config/crd/bases/platform.stackrox.io_securedclusters.yaml

RUN curl -L --retry 10 --silent --show-error --fail -o "/usr/local/bin/ocm" \
    https://github.com/openshift-online/ocm-cli/releases/download/v0.1.64/ocm-linux-amd64 && \
    chmod +x /usr/local/bin/ocm

ARG GOPATH=/go
ENV GOPATH=${GOPATH}

ARG GOCACHE=/go/.cache
ENV GOCACHE=${GOCACHE}

ARG GOROOT=/usr/local/go
ENV GOROOT=${GOROOT}

ARG GOFLAGS=-mod=mod
ENV GOFLAGS=${GOFLAGS}

RUN mkdir -p ${GOPATH}
WORKDIR ${GOPATH}
