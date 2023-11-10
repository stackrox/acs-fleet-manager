FROM registry.access.redhat.com/ubi8/ubi-minimal:8.8 as build
ENV HOME=/opt/helm
COPY rhacs-terraform  ${HOME}/rhacs-terraform
WORKDIR ${HOME}

RUN curl -L --retry 10 --silent --show-error --fail -o /usr/local/bin/helm \
    "https://mirror.openshift.com/pub/openshift-v4/clients/helm/latest/helm-linux-amd64" && \
    chmod +x /usr/local/bin/helm

RUN helm repo add external-secrets "https://charts.external-secrets.io/" && \
    helm dependencies build ${HOME}/rhacs-terraform

FROM quay.io/operator-framework/helm-operator:v1.32.0

ENV HOME=/opt/helm
COPY watches.yaml ${HOME}/watches.yaml
COPY --from=build ${HOME}/rhacs-terraform ${HOME}/rhacs-terraform
WORKDIR ${HOME}
