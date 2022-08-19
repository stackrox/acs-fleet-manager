FM_ENDPOINT="https://xtr6hh3mg6zc80v.api.stage.openshift.com"
CLUSTER_NAME="acs-stage-dp-01"
CLUSTER_ID=$(ocm list cluster "${CLUSTER_NAME}" --no-headers --columns="ID")

helm template rhacs-terraform \
  --debug \
  --namespace=rhacs \
  --values=/home/$USER/tmp_secrets/secrets.yaml \
  --set fleetshardSync.authType="RHSSO" \
  --set fleetshardSync.fleetManagerEndpoint=${FM_ENDPOINT} \
  --set fleetshardSync.clusterId=${CLUSTER_ID} \
  --set acsOperator.enabled=true .
