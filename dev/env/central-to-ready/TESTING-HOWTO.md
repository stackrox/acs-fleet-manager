# Transitioning a central to "ready" state

## Prepare

Reset DB:

```
$ make db/teardown; make db/setup; make db/migrate
```

Configure:

Make sure that the configuration file `config/dataplane-cluster-config.yaml` contains one of the available example configurations for a dataplane cluster. Otherwise any attempt to create a new central will be rejected by the fleet-manager.

Run the fleet-manager:

```
$ fleet-manager serve
```

(Wait until `clusters` table is populated with test cluster, e.g. Minikube.)

## Test

```
$ fmcurl rhacs/v1/agent-clusters/1234567890abcdef1234567890abcdef/centrals
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

After some reconciliations the central request should automatically transition to `provisioning`.

### Transition Dinosaur to ready

```
$ export CENTRAL_ID=$(fmcurl rhacs/v1/agent-clusters/1234567890abcdef1234567890abcdef/centrals | jq -r '.items[0].id')
$ fmcurl rhacs/v1/agent-clusters/1234567890abcdef1234567890abcdef/centrals/status -XPUT --data-binary @<(envsubst < central-status-update-ready.json)
$ fmcurl rhacs/v1/agent-clusters/1234567890abcdef1234567890abcdef/centrals/status -XPUT --data-binary @<(envsubst < central-status-update-ready.json)
```

Yes, this needs to be called twice to reflect fleetshard-sync's expected behaviour of periodically continuously calling fleet-manager.

Yields:

```
serviceapitests=# select * from dinosaur_requests ;
┌─[ RECORD 1 ]──────────────────────┬──────────────────────────────────────────────────────┐
│ id                                │ ca6e9bnafa3gja5a36pg                                 │
│ created_at                        │ 2022-05-24 13:59:10.918577+00                        │
│ updated_at                        │ 2022-05-24 14:04:45.452084+00                        │
│ deleted_at                        │                                                      │
│ region                            │ standalone                                           │
│ cluster_id                        │ 1234567890abcdef1234567890abcdef                     │
│ cloud_provider                    │ standalone                                           │
│ multi_az                          │ t                                                    │
│ name                              │ test1                                                │
│ status                            │ ready                                                │
│ subscription_id                   │                                                      │
│ owner                             │ mclasmei@redhat.com                                  │
│ owner_account_id                  │ 54188697                                             │
│ host                              │ foo                                                  │
│ organisation_id                   │ 11009103                                             │
│ failed_reason                     │                                                      │
│ placement_id                      │                                                      │
│ desired_dinosaur_version          │                                                      │
│ actual_dinosaur_version           │ 2.4.1                                                │
│ desired_dinosaur_operator_version │                                                      │
│ actual_dinosaur_operator_version  │ 0.21.2                                               │
│ dinosaur_upgrading                │ f                                                    │
│ dinosaur_operator_upgrading       │ f                                                    │
│ instance_type                     │ eval                                                 │
│ quota_type                        │ quota-management-list                                │
│ routes                            │ \x5b7b22446f6d61696e223a22746573742d726f7574652[...] │
│ routes_created                    │ t                                                    │
│ namespace                         │                                                      │
│ routes_creation_id                │                                                      │
└───────────────────────────────────┴──────────────────────────────────────────────────────┘
```
