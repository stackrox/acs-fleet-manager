#/bin/bash

set -o errexit   # -e
set -o pipefail
set -o nounset   # -u

# TODO: Create a driver script instead of implicitly connecting these with constants.
export INPUT_FILE_SECRETS="/var/tmp/tmp_secrets/secrets.yaml"  # See fetch_too_many_secrets.yaml
export OUTPUT_FILE="/var/tmp/tmp_secrets/acs-terraform-values.yaml"
export OUTPUT_DIRECTORY="$(dirname ${OUTPUT_FILE})"
export OUTPUT_FILE_TEMPLATE="${OUTPUT_DIRECTORY}/acs-terraform-values-template.yaml"


# TODO(ROX-12222): Replace Vault and BitWarden with Secret Management
CLUSTER_NAME="acs-stage-dp-01"
CLUSTER_ID=$(ocm list cluster "${CLUSTER_NAME}" --no-headers --columns="ID")

# Requires yq
# https://github.com/mikefarah/yq#install
# brew install yq  # Alternatively: sudo snap install yq

cat <<EOF > ${OUTPUT_FILE_TEMPLATE}
acsOperator:
  enabled: true
  source: rhacs-operators
  sourceNamespace: openshift-marketplace
  startingCSV: rhacs-operator.v3.71.0
fleetshardSync:
  authType: STATIC_TOKEN
  clusterId: 4ee13a50-871d-46b7-bc8b-fb101e02a37a
  createAuthProvider: true
  fleetManagerEndpoint: http://fleet-manager.rhacs.svc.cluster.local:8000
  image: quay.io/app-sre/acs-fleet-manager:main
  ocmToken: ""
  redHatSSO:
    clientId: OVERRIDE_ME_WITH_SECRET
    clientSecret: OVERRIDE_ME_WITH_SECRET
  tokenRefresher:
    image: quay.io/rhoas/mk-token-refresher:latest
    issuerUrl: https://sso.redhat.com/auth/realms/redhat-external
logging:
  aws:
    accessKeyId: OVERRIDE_ME_WITH_SECRET
    region: us-east-1
    secretAccessKey: OVERRIDE_ME_WITH_SECRET
  enabled: true
  global: {}
observability:
  enabled: true
  github:
    accessToken: OVERRIDE_ME_WITH_SECRET
    repository: https://api.github.com/repos/stackrox/rhacs-observability-resources/contents
    tag: master
  global: {}
  observabilityOperatorVersion: v3.0.14
  observatorium:
    authType: redhat
    gateway: https://observatorium-mst.api.stage.openshift.com
    metricsClientId: OVERRIDE_ME_WITH_SECRET
    metricsSecret: OVERRIDE_ME_WITH_SECRET
    redHatSsoAuthServerUrl: https://sso.redhat.com/auth/
    redHatSsoRealm: redhat-external
    tenant: rhacs
EOF

yq '. *= load(env(INPUT_FILE_SECRETS))' ${OUTPUT_FILE_TEMPLATE} > ${OUTPUT_FILE}

# See https://mikefarah.gitbook.io/yq/operators/multiply-merge
echo "Helm values file successfully written to ${OUTPUT_FILE}"