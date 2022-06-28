# fleetshard-sync

## Prerequisites

Start minikube, see environment specific setting e.g.  https://minikube.sigs.k8s.io/docs/drivers/:
```
$ minikube start
```

Start the RHACS operator:
```
$ cdrox
$ make -C operator install run
```

## Quickstart

Execute all commands from git root directory.

1. Start fleet manager:
    ```
    $ ./scripts/setup-dev-env.sh
    ```

1. Build and run fleetshard-sync:
    ```
    $ make fleetshard/build
    $ OCM_TOKEN=$(ocm token --refresh) CLUSTER_ID=1234567890abcdef1234567890abcdef ./fleetshard-sync
    ```

1. Create a central instance:
    ```
    $ ./scripts/create-central.sh
    ```

## Authentication types

Fleetshard sync provides different authentication types that can be used when calling the fleet manager's API:
- OCM refresh token
  - This will use the OCM refresh token obtained via `ocm token --refresh` and will be refreshed before expiring.
- RH SSO
  - This will use the client_credentials grant to obtain an access token. Additionally, it uses the [token-refresher](https://gitlab.cee.redhat.com/mk-ci-cd/mk-token-refresher)
    for obtaining new access tokens before expiring. Currently, the token-refresher is deployed via helm.
