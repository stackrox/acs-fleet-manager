set -exo pipefail

FM_ENDPOINT="https://xtr6hh3mg6zc80v.api.stage.openshift.com"
CLUSTER_NAME="acs-stage-dp-01"
CLUSTER_ID=$(ocm list cluster "${CLUSTER_NAME}" --no-headers --columns="ID")

# set +x
# ./fetch_too_many_secrets.sh
# set -x

# Alternatively:
#   --set acsOperator.source=redhat-operators
#   --set acsOperator.sourceNamespace=openshift-marketplace

# helm uninstall rhacs-terraform --namespace rhacs

# helm template ... to debug changes
helm install rhacs-terraform \
  --debug \
  --namespace rhacs \
  --create-namespace \
  --values=/home/$USER/tmp_secrets/secrets.yaml \
  --set fleetshardSync.authType="RHSSO" \
  --set fleetshardSync.fleetManagerEndpoint=${FM_ENDPOINT} \
  --set fleetshardSync.clusterId=${CLUSTER_ID} \
  --set acsOperator.enabled=true . \
  --set acsOperator.source=rhacs-operators \
  --set acsOperator.startingCSV=rhacs-operator.v3.71.0

# To delete all resources:
# helm template ... > /var/tmp/resources.yaml
# kubectl delete -f /var/tmp/resources.yaml
