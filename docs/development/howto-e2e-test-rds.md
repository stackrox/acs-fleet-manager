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
    # Prepare environment and secrets
    export PATH="$PATH:$(pwd)/bin"
    source ./scripts/lib/external_config.sh
    kinit # get a kerberos ticket
    export AWS_AUTH_HELPER=aws-saml
    init_chamber
    # When prompted select your profile for the dev AWS account arn:aws:iam::047735621815:role/047735621815-poweruser

    source <(run_chamber env "fleetshard-sync")
    source <(run_chamber env -b secretsmanager "fleetshard-sync")
    source <(run_chamber env "cluster-acs-dev-dp-01")
    export MANAGED_DB_ENABLED=true

    ./fleetshard-sync

    ```

1. Create a central instance and wait for DB Creation

    ```
    central_id=$(./scripts/create-centrals.sh | jq '.id' -r)
    # Watch the fleetshard-sync logs to tell what's happening in the background.
    # It should print something like this if everything works like expected:
    # RDS instance status: creating (instance ID: rhacs-chcb5m8ah6b2ko6qut0g-db-instance)
    # At somepoint fleetshard-sync will logs this:
    Unexpected error occurred rhacs-chcb5m8ah6b2ko6qut0g/test-central-1: getting Central DB connection string: initializing managed DB: initializing managed DB: beginning PostgreSQL transaction: dial tcp 10.1.206.163:5432: connect: operation timed out

    # This is expected because the RDS DB is created within a VPC you can't reach from your local environment, we assume that this test was a success, since the DB got to a avaialble state.
    ```

1. Make sure DB state is available and 2 instances exist in state available
1. Delete the central

    ```
    export OCM_TOKEN=$(ocm token)
    ./scripts/fmcurl "rhacs/v1/centrals/$central_id?async=true" -XDELETE  
    ```
