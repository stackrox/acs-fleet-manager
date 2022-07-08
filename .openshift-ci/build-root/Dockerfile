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
RUN curl -L --silent --show-error --fail -o "/usr/local/bin/yq" \
    https://github.com/mikefarah/yq/releases/download/v4.2.0/yq_linux_amd64 && \
    chmod +x /usr/local/bin/yq
RUN curl -L --silent --show-error --fail -O https://nodejs.org/dist/v16.15.1/node-v16.15.1-linux-x64.tar.xz && \
    mkdir -p /usr/local/lib/nodejs && \
    tar -xJvf node-v16.15.1-linux-x64.tar.xz -C /usr/local/lib/nodejs
RUN curl -L --silent --show-error --fail -o "/usr/local/bin/stern" \
    https://github.com/wercker/stern/releases/download/1.11.0/stern_linux_amd64 && \
    chmod +x /usr/local/bin/stern
ENV PATH="/usr/local/lib/nodejs/node-v16.15.1-linux-x64/bin:${PATH}"
