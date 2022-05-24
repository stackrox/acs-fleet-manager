# Transitioning a central to "ready" state

install olm:
$ brew install operator-sdk   # Install the operator SDK
$ operator-sdk olm install    # Install the OLM operator to your cluster
$ kubectl -n olm get pods -w  # Verify installation of OLM

install certmanager cluster:
$ kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.8.0/cert-manager.yaml

## setup operator (from stackrox/operator/README.md)

0. Make sure you have [cert-manager installed](https://cert-manager.io/docs/installation/).
   It takes care of the TLS aspects of the connection from k8s API server to the webhook server
   embedded in the manager binary.

1. Build operator image
   ```bash
   $ make docker-build
   ```
2. Make the image available for the cluster, this depends on k8s distribution you use.  
   You don't need to do anything when using KIND.  
   For minikube it could be done like this
   ```bash
   $ docker save stackrox/stackrox-operator:$(make tag) | ssh -o StrictHostKeyChecking=no -i $(minikube ssh-key) docker@$(minikube ip) docker load
   ```
3. Install CRDs and deploy operator resources
   ```bash
   $ make deploy
   ```
4. Validate that the operator's pod has started successfully
   ```bash
   $ kubectl -n stackrox-operator-system describe pods
   ```
   Check logs
   ```bash
   $ kubectl -n stackrox-operator-system logs deploy/rhacs-operator-controller-manager manager -f
   ```

FIX image in controller pod, wrong version information (0.0.1) instead of referencing newly build image?
....

reset
$ make db/teardown
$ make db/setup
$ make db/migrate

run
$ fleet-manager serve

## Test

$ fmcurl rhacs/v1/agent-clusters/1234567890abcdef1234567890abcdef/dinosaurs
=> empty list

$ fmcurl '/rhacs/v1/centrals?async=true' -XPOST --data-binary '@./central-request.json'
=> Central created

$ make db/psql

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

serviceapitests=# 

### transition to provisioning

serviceapitests=# update dinosaur_requests set status = 'provisioning';
UPDATE 1
serviceapitests=# 

serviceapitests=# update dinosaur_requests set host = 'foo';
UPDATE 1
serviceapitests=# 

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

### transition to ready

$ export ID=$(fmcurl rhacs/v1/agent-clusters/1234567890abcdef1234567890abcdef/dinosaurs | jq -r '.items[0].id')

$ fmcurl rhacs/v1/agent-clusters/1234567890abcdef1234567890abcdef/dinosaurs/status -XPUT --data-binary @<(envsubst < dinosaur-status-update-ready.json)

$ fmcurl rhacs/v1/agent-clusters/1234567890abcdef1234567890abcdef/dinosaurs/status -XPUT --data-binary @<(envsubst < dinosaur-status-update-ready.json)

Yes, I have to do the PUT twice, I don't know yet why this is the case.

But then I have:

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

serviceapitests=# 
