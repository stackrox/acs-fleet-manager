# About

This directory defines a docker compose based test environment which is easy to spin up.

## Preprequisites

1. A Docker daemon needs to be running and the `docker` CLI needs to be installed.
1. Setup [direnv](https://direnv.net/). Note: This is the recommended approach since `direnv` provides automatic environment loading and unloading for your shell, but this is not a strict requirement; if desired it is possible to load the required shell environments manually with `source`.

## Spinning up Test Environment

1. Make sure that no other DB is running:
    ```
    $ make -C $(git rev-parse --show-toplevel) db/teardown
    ```

1. Enter this directory:
    ```
    $ cd $(git rev-parse --show-toplevel)/dev/env/docker-compose
    ```
    Initial setup of `direnv` requires whitelisting of the provided test environment:
    ```
    $ direnv allow
    ```
    This will make sure that certain settings are available in the current environment, which are
    then picked up by `docker-compose.yaml`. In particular, this includes OCM tokens and secrets
    from `secrets/`. If the OCM token needs to be refreshed, use `direnv reload`.

1. Spin up the test environment using:
    ```
    $ docker compose up
    ```
    The `fleet-manager` service will be reachable at `localhost:8000`. Single data-plane cluster is configured
    as (see `config/dataplane-cluster-configuration.yaml` in this directory, not in the top-level directory of this repository).

1. If new Docker images for `fleet-manager` and `fleetshard-sync` shall be built, use:
    ```
    $ docker compose up --build
    ```
    instead.

1. To shut down the test environment:
    ```
    $ docker compose down
    ```

1. If the Postgres volume shall also be deleted:
    ```
    $ docker compose down -v
    ```

## Executing API calls

Within the test environment `fmcurl` can be used for quickly executing requests against `fleet-manager`'s API.
For example:
```
$ fmcurl rhacs/v1/agent-clusters/${CLUSTER_ID}/centrals
```
for listing all centrals associated with the configured test cluster.
