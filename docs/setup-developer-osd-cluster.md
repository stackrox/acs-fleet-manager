# How-To setup developer OSD cluster (step by step copy/paste guide)

### Pre-requirements

You will require several commands in order to use simple copy/paste.
1. `jq` and `yq` - JSON and YAML query CLI tools.
2. `bw` - BitWarden CLI. We need this to get values from BitWarden directly without paste/copy.
3. `ocm` - Openshift cluster manager CLI tool. We need it to create OSD cluster and manage it.
4. `oc` - Openshift cluster CLI tool (similar to kubectl). We need it to deploy resource into OSD cluster.
5. `ktunnel` - Reverse proxy to proxy service from kubernetes to local machine. You can find more info here: https://github.com/omrikiei/ktunnel
6. `watch` - (optional) To repeatedly executes specific command.
7. `grpcurl` - (optional) Requirement for execute gRPC calls.

Additionally, you will also require `quay.io` credentials.

### Intro

All commands should be executed in root directory of `stackrox/acs-fleet-manager` project.

### Create OSD Cluster

1. Create OSD Cluster with `ocm`

Export name for your cluster. Prefix it with your initials or something similar to avoid name collisions. i.e. `mt-osd-1307`
```
export OSD_CLUSTER_NAME="<your cluster name>"
```

For staging OSD cluster, you should login to staging platform. You should use `rhacs-managed-service-dev` account. The `ocm` command is aware of differences and defining `--url staging` all what is required in order to login to staging platform.
```
ocm login --url staging --token="<your token from OpenShift console UI - console.redhat.com>
```
Staging UI is accessible on this URL: https://qaprodauth.cloud.redhat.com

Create cluster with `ocm` command
```
# Get AWS Keyes from BitWarden
export AWS_REGION="us-east-1"
export AWS_ACCOUNT_ID=$(bw get item "23a0e6d6-7b7d-44c8-b8d0-aecc00e1fa0a" | jq '.fields[] | select(.name | contains("AccountID")) | .value' --raw-output)
export AWS_ACCESS_KEY_ID=$(bw get item "23a0e6d6-7b7d-44c8-b8d0-aecc00e1fa0a" | jq '.fields[] | select(.name | contains("AccessKeyID")) | .value' --raw-output)
export AWS_SECRET_ACCESS_KEY=$(bw get item "23a0e6d6-7b7d-44c8-b8d0-aecc00e1fa0a" | jq '.fields[] | select(.name | contains("SecretAccessKey")) | .value' --raw-output)

# Execute creation command
ocm create cluster \
  --ccs \
  --aws-access-key-id "${AWS_ACCESS_KEY_ID}" \
  --aws-account-id "${AWS_ACCOUNT_ID}" \
  --aws-secret-access-key "${AWS_SECRET_ACCESS_KEY}" \
  --region "${AWS_REGION}" \
  --multi-az \
  --compute-machine-type "m5a.xlarge" \
  --version "4.10.20" \
  "${OSD_CLUSTER_NAME}"
```

You will see output of command. Output should contain "ID" of the cluster. Export that ID to `CLUSTER_ID` environment variable.
```
export CLUSTER_ID="<ID of the cluster>"
```

Now, you have to wait for cluster to be provisioned. Check status of cluster creation:
```
watch --interval 10 ocm cluster status ${CLUSTER_ID}
```

2. Add auth provider for OSD cluster

This is required in order to be able to log-in to cluster. In UI or with `oc` command. You can pick your own admin pass, here we use `md5`.
If you need password for UI login, be sure to store it somewhere.
```
export OSD_ADMIN_USER="osd-admin"
export OSD_ADMIN_PASS=$(date | md5)

echo "{\"htpasswd\":{\"password\":\"${OSD_ADMIN_PASS}\",\"username\":\"${OSD_ADMIN_USER}\"},\"login\":true,\"mapping_method\":\"add\",\"name\":\"osd-htpasswd\",\"type\":\"HTPasswdIdentityProvider\"}" > ./tmp-osd-htpasswd-body.txt
ocm post "/api/clusters_mgmt/v1/clusters/${CLUSTER_ID}/identity_providers" --body ./tmp-osd-htpasswd-body.txt

echo "{\"id\":\"${OSD_ADMIN_USER}\"}" > ./tmp-osd-cluster-admins-body.txt
ocm post "/api/clusters_mgmt/v1/clusters/${CLUSTER_ID}/groups/cluster-admins/users" --body ./tmp-osd-cluster-admins-body.txt
```

3. Login to OSD cluster with `oc` command
```
export CLUSTER_API_URL=$(ocm get "/api/clusters_mgmt/v1/clusters/${CLUSTER_ID}" | jq '.api.url' --raw-output)
oc login "${CLUSTER_API_URL}" --username=${OSD_ADMIN_USER} --password=${OSD_ADMIN_PASS}
```
If login step fails, it can be the case that previously created auth provider and user are not applied yet on the cluster. You can wait few seconds and try again.

### Prepare cluster for RHACS Operator

4. Export defaults
```
export RHACS_OPERATOR_CATALOG_VERSION="3.70.1"
export RHACS_OPERATOR_CATALOG_NAME="redhat-operators"
```

5. Check if the latest version of available ACS Operator is high enough for you. If that is OK for you, you can skip next steps prefixed with `(ACS operator from branch)`.

Execute the following command in separate terminal (new shell).
```
oc port-forward -n openshift-marketplace svc/redhat-operators 50051:50051
```

```
grpcurl -plaintext -d '{"name":"rhacs-operator"}' localhost:50051 api.Registry/GetPackage | jq '.channels[0].csvName'
```
You can stop port-forward after this.

6. (ACS operator from branch) Prepare pull secret
**Important** This will change cluster wide pull secrets. It's not advised to use on clusters where credentials can be compromized.

**Pay attention:** `docker-credential-osxkeychain` is specific for MacOS. For Linux please check `docker-credential-secretservice`.
```
export QUAY_REGISTRY_AUTH_BASIC=$(docker-credential-osxkeychain get <<<"https://quay.io" | jq -r '"\(.Username):\(.Secret)"')

oc get secret/pull-secret -n openshift-config --template='{{index .data ".dockerconfigjson" | base64decode}}' > ./tmp-pull-secret.json
oc registry login --registry="quay.io/rhacs-eng" --auth-basic="${QUAY_REGISTRY_AUTH_BASIC}" --to=./tmp-pull-secret.json
oc set data secret/pull-secret -n openshift-config --from-file=.dockerconfigjson=./tmp-pull-secret.json
```

7. (ACS operator from branch) Deploy catalog

You should find catalog build from your branch or from master branch of `stackrox/stackrox` repository. You should look at CircleCI job with name `build-operator` and step `Build and push images for quay.io/rhacs-eng`. In log, you can find image tag. Something like `v3.71.0-16-g3f8fcd60c6`. Export that value without `v`
```
export RHACS_OPERATOR_CATALOG_VERSION="<Stackrox Operator Index version>"
```

Run the following command to register new ACS Observability operator catalog.
```
export RHACS_OPERATOR_CATALOG_NAME="rhacs-operators"

oc apply -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: ${RHACS_OPERATOR_CATALOG_NAME}
  namespace: openshift-marketplace
spec:
  displayName: 'RHACS Development'
  publisher: 'Red Hat ACS'
  sourceType: grpc
  image: quay.io/rhacs-eng/stackrox-operator-index:v${RHACS_OPERATOR_CATALOG_VERSION}
EOF
```

By executing:
```
oc get pods -n openshift-marketplace
```
You should be able to see `rhacs-operators` pod running.

### Terraform OSD cluster with Fleet Synchronizer

8. Export defaults
```
# Copy static token from BitWarden
export STATIC_TOKEN=$(bw get item "64173bbc-d9fb-4d4a-b397-aec20171b025" | jq '.fields[] | select(.name | contains("JWT")) | .value' --raw-output)

export FLEET_MANAGER_IMAGE=quay.io/app-sre/acs-fleet-manager:main
export STARTING_CSV="rhacs-operator.v3.70.1"
```

9. Prepare namespace
```
export NAMESPACE=rhacs
export FLEET_MANAGER_ENDPOINT="http://fleet-manager.${NAMESPACE}.svc.cluster.local:8000"

oc create namespace "${NAMESPACE}"
```

10. (Optional local fleet synchronizer build) Build and push fleet synchronizer

```
export IMAGE_TAG=osd-test

GOARCH=amd64 GOOS=linux CGO_ENABLED=0 make image/build/push/internal
```

11. (Optional local fleet synchronizer build) Get Fleet Manager image name
```
export FLEET_MANAGER_IMAGE=$(oc get route default-route -n openshift-image-registry -o jsonpath="{.spec.host}")/${NAMESPACE}/fleet-manager:${IMAGE_TAG}
export STARTING_CSV="rhacs-operator.v${RHACS_OPERATOR_CATALOG_VERSION}"
```

12. Terraform cluster
```
helm upgrade --install rhacs-terraform \
  --namespace "${NAMESPACE}" \
  --set fleetshardSync.authType="STATIC_TOKEN" \
  --set fleetshardSync.image="${FLEET_MANAGER_IMAGE}" \
  --set fleetshardSync.fleetManagerEndpoint="${FLEET_MANAGER_ENDPOINT}" \
  --set fleetshardSync.staticToken="${STATIC_TOKEN}" \
  --set fleetshardSync.clusterId="${CLUSTER_ID}" \
  --set acsOperator.enabled=true \
  --set acsOperator.source="rhacs-operators" \
  --set acsOperator.startingCSV="${STARTING_CSV}" \
  --set observability.enabled=false ./dp-terraform/helm/rhacs-terraform
```

13. Create tunnel from cluster to local machine

Execute the following command in separate terminal (new shell). Ensure that you have same namespace as one defined in `$NAMESPACE`.
```
export NAMESPACE=rhacs

ktunnel expose --namespace "${NAMESPACE}" fleet-manager 8000:8000 --reuse
```

### Setup local Fleet Manager

14. Create OSD Cluster config file for fleet manager

Ensure that you are in correct kube context.
```
export OC_CURRENT_CONTEXT=$(oc config current-context)
export OSD_CLUSTER_DOMAIN=$(ocm get /api/clusters_mgmt/v1/clusters/${CLUSTER_ID} | jq '.dns.base_domain' --raw-output)

cat << EOF > "./${CLUSTER_ID}.yaml"
---
clusters:
 - name: '${OC_CURRENT_CONTEXT}'
   cluster_id: '${CLUSTER_ID}'
   cloud_provider: aws
   region: ${AWS_REGION}
   schedulable: true
   status: ready
   multi_az: true
   central_instance_limit: 10
   provider_type: standalone
   supported_instance_type: "eval,standard"
   cluster_dns: '${OSD_CLUSTER_NAME}.${OSD_CLUSTER_DOMAIN}'
   available_central_operator_versions:
     - version: "${RHACS_OPERATOR_CATALOG_VERSION}"
       ready: true
       central_versions:
         - version: "${RHACS_OPERATOR_CATALOG_VERSION}"
EOF
```

15. Build, setup and start local fleet manager

Execute the following command in separate terminal (new shell). Ensure that you have same exported `CLUSTER_ID`.
```
# Build binary
make binary

# Setup DB
make db/teardown db/setup db/migrate

# Start local fleet manager
./fleet-manager serve --dataplane-cluster-config-file "./${CLUSTER_ID}.yaml"
```

### Install central

16. Prepare default values
```
# Copy static token from BitWarden
export STATIC_TOKEN=$(bw get item "64173bbc-d9fb-4d4a-b397-aec20171b025" | jq '.fields[] | select(.name | contains("JWT")) | .value' --raw-output)

export AWS_REGION="us-east-1"
```

17. Call curl to install central

```
export CENTRAL_ID=$(curl --location --request POST "http://localhost:8000/api/rhacs/v1/centrals?async=true" --header "Content-Type: application/json" --header "Accept: application/json" --header "Authorization: Bearer ${STATIC_TOKEN}" --data-raw "{\"name\":\"test-on-cluster\",\"cloud_provider\":\"aws\",\"region\":\"${AWS_REGION}\",\"multi_az\":true}" | jq '.id' --raw-output)
```

18. Check if new namespace is created and if all pods are up and running
```
export CENTRAL_NAMESPACE="${NAMESPACE}-${CENTRAL_ID}"

oc get pods --namespace "${CENTRAL_NAMESPACE}"
```

### Install sensor to same data plane cluster where central is installed

19. Fetch sensor configuration
```
export ROX_ADMIN_PASSWORD=$(oc get secrets -n "${CENTRAL_NAMESPACE}" central-htpasswd -o yaml | yq .data.password | base64 --decode)
roxctl sensor generate openshift --openshift-version=4 --endpoint "https://central-${CENTRAL_NAMESPACE}.apps.${OSD_CLUSTER_NAME}.${OSD_CLUSTER_DOMAIN}:443" --insecure-skip-tls-verify -p "${ROX_ADMIN_PASSWORD}" --admission-controller-listen-on-events=false --disable-audit-logs=true --central="https://central-${CENTRAL_NAMESPACE}.apps.${OSD_CLUSTER_NAME}.${OSD_CLUSTER_DOMAIN}:443" --collection-method=none --name osd-cluster-sensor
```

20. Install sensor

This step requires `quay.io` username and password. Have that prepared.
```
./sensor-osd-cluster-sensor/sensor.sh
```

21. Check that sensor is up and running

Sensor uses `stackrox` namespace by default.
```
oc get pods -n stackrox
```

### Extend OSD cluster lifetime to 7 days

By default, staging cluster will be up for 2 days. You can extend it to 7 days. To do that, execute the following command for MacOS:
```
echo "{\"expiration_timestamp\":\"$(date --iso-8601=seconds -d '+7 days')\"}" | ocm patch "/api/clusters_mgmt/v1/clusters/${CLUSTER_ID}"
```

Or on Linux:
```
echo "{\"expiration_timestamp\":\"$(date -v+7d -u +'%Y-%m-%dT%H:%M:%SZ')\"}" | ocm patch "/api/clusters_mgmt/v1/clusters/${CLUSTER_ID}"
```

### Re-deploy new Fleetshard synchronizer

To deploy a new build of Fleetshard synchronizer, you can simply re-build and push the image and after that rollout restart of deployment is sufficient.
```
GOARCH=amd64 GOOS=linux CGO_ENABLED=0 make image/build/push/internal
oc rollout restart -n "${NAMESPACE}" deployment fleetshard-sync
```

### Re-start new local Fleetshard manager

```
make binary
./fleet-manager serve --dataplane-cluster-config-file "./${CLUSTER_ID}.yaml"
```
