# Transitioning a central to "ready" state

## Prepare

Reset DB:

```
$ make db/teardown; make db/setup; make db/migrate
```

Run:

```
$ fleet-manager serve
```

(Wait until `clusters` table is populated with test cluster, e.g. Minikube.)

## Test

```
$ fmcurl rhacs/v1/agent-clusters/1234567890abcdef1234567890abcdef/dinosaurs
```
=> empty list

```
$ fmcurl '/rhacs/v1/centrals?async=true' -XPOST --data-binary '@./central-request.json'
```
=> Central created

```
$ make db/psql
[...]
serviceapitests=# select * from dinosaur_requests ;
┌─[ RECORD 1 ]──────────────────────┬──────────────────────────────────┐
│ id                                │ ca6duifafa3g1gsiu3jg             │
│ created_at                        │ 2022-05-24 13:36:09.416818+00    │
│ updated_at                        │ 2022-05-24 13:36:09.416818+00    │
│ deleted_at                        │                                  │
│ region                            │ standalone                       │
│ cluster_id                        │ 1234567890abcdef1234567890abcdef │
│ cloud_provider                    │ standalone                       │
│ multi_az                          │ t                                │
│ name                              │ test1                            │
│ status                            │ accepted                         │
│ subscription_id                   │                                  │
│ owner                             │ mclasmei@redhat.com              │
│ owner_account_id                  │ 54188697                         │
│ host                              │                                  │
│ organisation_id                   │ 11009103                         │
│ failed_reason                     │                                  │
│ placement_id                      │                                  │
│ desired_dinosaur_version          │                                  │
│ actual_dinosaur_version           │                                  │
│ desired_dinosaur_operator_version │                                  │
│ actual_dinosaur_operator_version  │                                  │
│ dinosaur_upgrading                │ f                                │
│ dinosaur_operator_upgrading       │ f                                │
│ instance_type                     │ eval                             │
│ quota_type                        │ quota-management-list            │
│ routes                            │                                  │
│ routes_created                    │ f                                │
│ namespace                         │                                  │
│ routes_creation_id                │                                  │
└───────────────────────────────────┴──────────────────────────────────┘
```

### Transition Dinosaur to provisioning

```
PGPASSWORD=$(cat secrets/db.password) psql -h localhost -d $(cat secrets/db.name) -U $(cat secrets/db.user) -f dev/env/central-to-ready/prepare-dinosaur-request-entry.sql
```

```
$ fmcurl rhacs/v1/agent-clusters/1234567890abcdef1234567890abcdef/dinosaurs
=>
{
  "kind": "ManagedDinosaurList",
  "items": [
    {
      "id": "ca6duifafa3g1gsiu3jg",
      "kind": "ManagedDinosaur",
      "metadata": {
        "name": "test1",
        "annotations": {
          "mas/id": "ca6duifafa3g1gsiu3jg",
          "mas/placementId": ""
        }
      },
      "spec": {
        "endpoint": {},
        "versions": {},
        "deleted": false
      }
    }
  ]
}
```

### Transition Dinosaur to ready

```
$ export ID=$(fmcurl rhacs/v1/agent-clusters/1234567890abcdef1234567890abcdef/dinosaurs | jq -r '.items[0].id')
$ fmcurl rhacs/v1/agent-clusters/1234567890abcdef1234567890abcdef/dinosaurs/status -XPUT --data-binary @<(envsubst < dinosaur-status-update-ready.json)
$ fmcurl rhacs/v1/agent-clusters/1234567890abcdef1234567890abcdef/dinosaurs/status -XPUT --data-binary @<(envsubst < dinosaur-status-update-ready.json)
```

Yes, this needs to be called twice to reflect fleetshard-sync's expected behaviour of periodically continuously calling fleet-manager.

Yields:

```
serviceapitests=# select * from dinosaur_requests ;
┌─[ RECORD 1 ]──────────────────────┬──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ id                                │ ca6e9bnafa3gja5a36pg                                                                                                         │
│ created_at                        │ 2022-05-24 13:59:10.918577+00                                                                                                │
│ updated_at                        │ 2022-05-24 14:04:45.452084+00                                                                                                │
│ deleted_at                        │                                                                                                                              │
│ region                            │ standalone                                                                                                                   │
│ cluster_id                        │ 1234567890abcdef1234567890abcdef                                                                                             │
│ cloud_provider                    │ standalone                                                                                                                   │
│ multi_az                          │ t                                                                                                                            │
│ name                              │ test1                                                                                                                        │
│ status                            │ ready                                                                                                                        │
│ subscription_id                   │                                                                                                                              │
│ owner                             │ mclasmei@redhat.com                                                                                                          │
│ owner_account_id                  │ 54188697                                                                                                                     │
│ host                              │ foo                                                                                                                          │
│ organisation_id                   │ 11009103                                                                                                                     │
│ failed_reason                     │                                                                                                                              │
│ placement_id                      │                                                                                                                              │
│ desired_dinosaur_version          │                                                                                                                              │
│ actual_dinosaur_version           │ 2.4.1                                                                                                                        │
│ desired_dinosaur_operator_version │                                                                                                                              │
│ actual_dinosaur_operator_version  │ 0.21.2                                                                                                                       │
│ dinosaur_upgrading                │ f                                                                                                                            │
│ dinosaur_operator_upgrading       │ f                                                                                                                            │
│ instance_type                     │ eval                                                                                                                         │
│ quota_type                        │ quota-management-list                                                                                                        │
│ routes                            │ \x5b7b22446f6d61696e223a22746573742d726f7574652d7072656669782d666f6f222c22526f75746572223a22636c75737465722e6c6f63616c227d5d │
│ routes_created                    │ t                                                                                                                            │
│ namespace                         │                                                                                                                              │
│ routes_creation_id                │                                                                                                                              │
└───────────────────────────────────┴──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Central to ready using an OSD cluster and only with API calls

```bash
# Assuming KUBECONFIG is set for the cluster
make db/teardown db/setup db/migrate
make binary && ./fleet-manager serve \
   --dataplane-cluster-config-file=$(pwd)/dev/config/dataplane-cluster-configuration-infractl-osd.yaml \
   --providers-config-file=$(pwd)/dev/config/provider-configuration-infractl-osd.yaml \
   --kubeconfig=${KUBECONFIG} \
   2>&1 | tee fleet-manager-serve.log

# should return 200 and no central
curl -v -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/rhacs/v1/centrals | jq .

cluster_id=$(grep cluster_id $(pwd)/dev/config/dataplane-cluster-configuration-infractl-osd.yaml | tail -n1 | tr -s ' '  | cut -d ' ' -f 3)
# wait for leader election
# check cluster is there: should return 200
curl -v -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/rhacs/v1/agent-clusters/${cluster_id} | jq .

# register operator version for cluster 
curl -v -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/rhacs/v1/agent-clusters/${cluster_id}/status?async=true -X PUT -d '
{
  "centralOperator": [
    {
      "ready": true,
      "version": "0.1.0",
      "centralVersions": ["0.1.0"]
    } 
  ],
  "conditions": [
    {
      "type": "Ready",
      "status": "true",
      "message": "fully operational",
      "reason": "this is a test"
    }
  ]
}' | jq .

# create a central for the same cloud provider and region as the cluster
curl -v -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/rhacs/v1/centrals?async=true -X POST -d '{"name": "foo-central", "multi_az": true, "cloud_provider": "aws", "region": "us-west-2"}' | jq .

# Central eventually gets to accepted state
curl -v -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/rhacs/v1/centrals | jq .

central_id=$(curl -v -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/rhacs/v1/centrals | jq -r '.items[0].id')
cluster_dns=$(grep cluster_dns $(pwd)/dev/config/dataplane-cluster-configuration-infractl-osd.yaml | tail -n1 | tr -s ' '  | cut -d ' ' -f 3)
# Simulate fleetshard-sync moving to ready
# this needs to be called twice to reflect fleetshard-sync's expected behaviour of periodically continuously calling fleet-manager.
for i in $(seq 2)
do
    curl -v -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/rhacs/v1/agent-clusters/${cluster_id}/centrals/status -X PUT -d "
    {
    \"${central_id}\": {
        \"conditions\": [
            {
                \"type\": \"Ready\",
                \"reason\": \"CentralInstanceReady\",
                \"message\": \"Installing\",
                \"status\": \"True\",
                \"lastTransitionTime\": \"2018-01-01T00:00:00Z\"
            }
        ],
        \"routes\": [
            {
                \"name\": \"test-route-name\",
                \"prefix\": \"test-route-prefix\",
                \"router\": \"${cluster_dns}\"
            }
        ],
        \"versions\": {
            \"dinosaur\": \"0.1.0\",
            \"dinosaurOperator\": \"0.1.0\"
        }
       }
    }
    " | jq .
    sleep 30
done

# central should be in ready state
curl -v -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/rhacs/v1/centrals | jq .
```