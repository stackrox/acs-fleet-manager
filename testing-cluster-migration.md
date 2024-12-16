# Testing clusterreassignment on multicluster

- Create 2 infra OSD on AWS clusters via UI (or infraclt)
- Login to cluster in different shells using KUBECONFIG variable
```bash
mkdir ~/kubes
# Get the kubectl URL from infractl artificats
url=$(infractl artifacts jm-migration-1 --json | jq '.Artifacts[] | select(.Name=="kubeconfig") | .URL' -r)
wget -O ~/kubes/cluster1 $url

url=$(infractl artifacts jm-migration-2 --json | jq '.Artifacts[] | select(.Name=="kubeconfig") | .URL' -r)
wget -O ~/kubes/cluster2 $url

# In different shells
export KUBECONFIG="$HOME/kubes/cluster1" # shell 1
export KUBECONFIG="$HOME/kubes/cluster2" # shell 2
```
- Install all components to cluster 1
```bash
# Shell with cluster1 kubeconfig
export CLUSTER_TYPE=infra-openshift
make deploy/bootstrap
make deploy/dev
# There is a known bug that FS sometimes does not pick up the pull secret and thus
# cannot use the cluster local pullsecret, deleting the pod can help

# Make sure the cluster has pull secret for tenant images (at least central) currently the fast stream operator version used does not have correct scanner images configured unfortunately
export quay_pw="$yourpw"
oc get secret/pull-secret -n openshift-config --template='{{index .data ".dockerconfigjson" | base64decode}}' > /tmp/pull
oc registry login --registry="quay.io/rhacs-eng" --auth-basic="jmalsam:$quay_pw" --to=/tmp/pull
oc set data secret/pull-secret -n openshift-config --from-file=.dockerconfigjson=/tmp/pull
```
- Make sure FM is available to public on a route on cluster1
```bash
# currently you have to set route termination to edge instead of reencrypt for the fleet-manager route
k patch -n rhacs route fleet-manager -p '{"spec":{"tls":{"termination":"edge"}}}'
export FM_URL="https://$(k get routes -n rhacs fleet-manager -o yaml | yq .spec.host)"

ocm login --use-device-code --url prod
# Verify you can call FM
export OCM_TOKEN=$(ocm token)
./scripts/fmcurl "rhacs/v1/centrals" -XGET -v
```
- Configure FM on cluster1 to additonally register cluster2 in it's clusterconfig
```bash
# Get the cluster-list config map with cluster1 kubeconfig
k get cm -n rhacs fleet-manager-dataplane-cluster-scaling-config -o yaml > dataplane-config.yaml
cat dataplane-config.yaml | yq '.data."dataplane-cluster-configuration.yaml"' > cluster-list.json

# using cluster2 clusterconfig
make cluster-list | jq '.[0] | .name="dev2" | .cluster_id="1234567890abcdef1234567890abcdeg"' | jq --slurp . > cluster-list2.json
jq --slurp '. | add' cluster-list.json cluster-list2.json -c

# Copy that string to the dataplane-cluster-configuration.yaml data in the configmap and apply
k apply -f dataplane-config.yaml
# Restart fleet-manager
k delete pod -n rhacs -l app=fleet-manager

# Verify new cluster was registered
k logs -n rhacs $FMPOD -c service | grep 'ready cluster'
```
- Install data plane stack to 2nd cluster
```bash
# Using the shell with cluster 2 config
# We are first going to install everything
# Then remove the FM installation and point FS to the FM of cluster1
export CLUSTER_TYPE=infra-openshift
make deploy/bootstrap
make deploy/dev
k delete deploy fleet-manager -n rhacs

# Use the first cluster1 to generate a static token for the 2nd cluster
STATIC_TOKEN=$(kubectl create token -n rhacs fleetshard-sync --audience acs-fleet-manager-private-api --duration 8760h)


# On the shell connected to cluster2 confige FS to reach out to the cluster1 FM
k patch fleetshards -n rhacs rhacs-terraform --type='merge' -p "{\"spec\":{\"fleetshardSync\":{\"authType\":\"STATIC_TOKEN\",\"staticToken\":\"$STATIC_TOKEN\",\"fleetManagerEndpoint\":\"$FM_URL\",\"clusterId\":\"1234567890abcdef1234567890abcdeg\"}}}"

# Verify fleetshard-sync on cluster2 is connected to the FM running on cluster 1
k logs -n rhacs fleetshard-sync
```
- Activate migration feature on all deployments
```bash
# Cluster1 shell
kubectl patch deploy -n rhacs fleetshard-sync -p '{"spec":{"template":{"spec":{"containers":[{"name":"fleetshard-sync","env":[{"name":"RHACS_CLUSTER_MIGRATION", "value":"true"}]}]}}}}'
kubectl patch deploy -n rhacs fleet-manager -p '{"spec":{"template":{"spec":{"containers":[{"name":"service","env":[{"name":"RHACS_CLUSTER_MIGRATION", "value":"true"}]}]}}}}'
# Activate routes for FM as well, set enable-central-external-certificate=true in commands
k edit deploy -n rhacs fleet-manager

# Cluster2 shell
kubectl patch deploy -n rhacs fleetshard-sync -p '{"spec":{"template":{"spec":{"containers":[{"name":"fleetshard-sync","env":[{"name":"RHACS_CLUSTER_MIGRATION", "value":"true"}]}]}}}}'
```
- Create a central tenant
```
export OCM_TOKEN=$(ocm token)
tid=$(./scripts/create-central.sh | jq .id -r)
# Should be schedule to cluster1 watch it beeing created
k get pods -n rhacs-$tid
# Verify as well that CNAME records were properly created
# Once created trigger a cluster migration
rhoas login --auth-url=https://auth.redhat.com/auth/realms/EmployeeIDP
export OCM_TOKEN=$(rhoas authtoken)
./scripts/fmcurl "rhacs/v1/admin/centrals/$tid/assign-cluster" -XPOST -d'{"cluster_id": "1234567890abcdef1234567890abcdeg"}' -v
# Watch the tenant disappear on cluster 1 and reappear on cluster 2
# Also verify that the DNS entry has changed for that tenant from router-default.apps.<cluster1basedns> to router-default.apps.<cluster2basedns>
```