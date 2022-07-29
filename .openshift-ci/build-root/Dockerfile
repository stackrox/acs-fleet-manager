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

FROM registry.ci.openshift.org/openshift/release:golang-1.17
RUN curl -L --retry 10 --silent --show-error --fail -o "/usr/local/bin/yq" \
    https://github.com/mikefarah/yq/releases/download/v4.2.0/yq_linux_amd64 && \
    chmod +x /usr/local/bin/yq
RUN curl -L --retry 10 --silent --show-error --fail -O https://nodejs.org/dist/v16.15.1/node-v16.15.1-linux-x64.tar.xz && \
    mkdir -p /usr/local/lib/nodejs && \
    tar -xJvf node-v16.15.1-linux-x64.tar.xz -C /usr/local/lib/nodejs && \
    rm node-v16.15.1-linux-x64.tar.xz
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
ENV PATH="/usr/local/lib/nodejs/node-v16.15.1-linux-x64/bin:${PATH}"
