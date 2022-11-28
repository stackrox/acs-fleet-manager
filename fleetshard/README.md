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

1. Bring up the environment
   ```shell
   make deploy/bootstrap deploy/dev
   ```
1. Create a central instance:
    ```
    $ ./scripts/create-central.sh
    ```

Also refer to [this guide](../docs/development/setup-test-environment.md) for more information

## Build binary
```shell
make fleetshard-sync
```

## External configuration
To run Fleetshard-sync locally, you may need to download the development configuration from AWS Parameter Store:
```shell
export AWS_AUTH_HELPER=aws-vault
source ./scripts/lib/external_config.sh
init_chamber
```
Dev environment is selected by default. After this you may call
```shell
run_chamber exec fleetshard-sync -- ./fleetshard-sync
```
to inject the necessary environment variables to the fleetshard-sync application.

## Authentication types

Fleetshard sync provides different authentication types that can be used when calling the fleet manager's API.

### Red Hat SSO

This is the default authentication type used.
To run fleetshard-sync with RH SSO, use the following command:
```shell
run_chamber exec fleetshard-sync -- ./fleetshard-sync
```

### OCM Refresh token

To run fleetshard-sync with the OCM refresh token, use the following:
```shell
OCM_TOKEN=$(ocm token --refresh) \
AUTH_TYPE=OCM \
run_chamber exec fleetshard-sync -- ./fleetshard-sync
```

### Static token

A static token has been created which is non-expiring. The JWKS certs are by default added to fleet manager.
The token's claims can be viewed under `config/static-token-payload.json`.
You can either generate your own token following the documentation under `docs/acs/test-locally-static-token.md` or
use the token found within Bitwarden (`ACS Fleet* static token`):
```
STATIC_TOKEN=<generated value | bitwarden value> \
AUTH_TYPE=STATIC_TOKEN \
run_chamber exec fleetshard-sync -- ./fleetshard-sync
```
