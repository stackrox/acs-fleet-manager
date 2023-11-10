FROM quay.io/operator-framework/helm-operator:v1.32.0

ENV HOME=/opt/helm
COPY watches.yaml ${HOME}/watches.yaml
COPY rhacs-terraform  ${HOME}/rhacs-terraform
WORKDIR ${HOME}
