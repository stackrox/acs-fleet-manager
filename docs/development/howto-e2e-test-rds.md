# How to e2e test RDS

At the point in time this documentation was written AWS RDS DB creation and deletion is not e2e tested with a full setup of fleet-manager and fleetshard-sync. Everytime a change to the RDS provisioning logic is introduced we need to e2e test that change manually using the steps described here.

**Prerequisites:**

- A K8s cluster to create central resources on (using CRC as an example here)
- Kubeconfig configured with access to that cluster
- Setup personal AWS access through `aws-saml.py` (see [secret-management.md](./secret-management.md))
- RHACS Operator running or installed in the cluster

1. Run local fleet-manager

    ```
    make db/teardown db/setup db/migrate

    make binary

    ./fleet-manager serve --dataplane-cluster-config-file ./dev/config
    ```

1. Run local fleetshard-sync

    ```
    # Prepare environment
    export AWS_AUTH_HELPER=aws-saml
    export MANAGED_DB_ENABLED=true
    export CLUSTER_NAME=local_cluster
    # flip the PubliclyAccessible flag to true in rds.go line 514
    make binary

    ./dev/env/scripts/exec_fleetshard_sync.sh
    ```

1. Create a central instance and wait for DB Creation

    ```
    central_id=$(./scripts/create-centrals.sh | jq '.id' -r)
    # Watch the fleetshard-sync logs to tell what's happening in the background.
    # It should print something like this if everything works like expected:
    # RDS instance status: creating (instance ID: rhacs-chcb5m8ah6b2ko6qut0g-db-instance)

    # At some point your central instance should become ready
    ```

1. Make sure DB state is available and 2 instances exist in state available the central pod is ready
1. Delete the central

    ```
    export OCM_TOKEN=$(ocm token)
    ./scripts/fmcurl "rhacs/v1/centrals/$central_id?async=true" -XDELETE  
    ```
