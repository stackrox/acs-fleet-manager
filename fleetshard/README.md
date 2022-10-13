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
   make deploy/apps
   ```
1. Create a central instance:
    ```
    $ ./scripts/create-central.sh
    ```

Also refer to [this guide](../docs/development/setup-test-environment.md) for more information

## Authentication types

Fleetshard sync provides different authentication types that can be used when calling the fleet manager's API.

### OCM Refresh token

This is the default authentication type used.
To run fleetshard-sync with the OCM refresh token, use the following:
```
$ OCM_TOKEN=$(ocm token --refresh) AUTH_TYPE=OCM ./fleetshard-sync
```

### Red Hat SSO

This will use the client_credentials grant to obtain an access token.
The access token will be obtained via the [token-refresher](https://gitlab.cee.redhat.com/mk-ci-cd/mk-token-refresher).

The token-refresher is deployed only within the Helm-based deployment:
```
$ helm install \
  --set fleetshardSync.authType=RHSSO \
  --set fleetshardSync.redhatSSO.clientId=<client-id> \
  --set fleetshardSync.redhatSSO.clientSecret=<client-secret> \
  fleetshard dp-terraform/helm/rhacs-terraform
```

If you want to test it locally, you can do the following with a token in a file:
```
$ http --form --auth <client-id>:<client-secret> POST https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/token grant_type=client_credentials > path/to/token/file
$ AUTH_TYPE=RHSSO RHSSO_TOKEN_FILE=path/to/token/file ./fleetshard-sync
```

This will have the disadvantage of the token expiring, you can also deploy the token-refresher image locally:
```
docker run -it \
   -v /path/to/your/token:/rhsso-token/ \
   quay.io/rhoas/mk-token-refresher:latest \
   --oidc.client-id=<rhsso-client-id> --oidc.client-secret=<rhsso-client-secret> \
   --oidc.issuer-url=https://sso.redhat.com/auth/realms/redhat-external --margin=1m --file=/rhsso-token/token  
$ AUTH_TYPE=RHSSO RHSSO_TOKEN_FILE=/path/to/token/token ./fleetshard-sync
```

### Static token

A static token has been created which is non-expiring. The JWKS certs are by default added to fleet manager.
The token's claims can be viewed under `config/static-token-payload.json`.
You can either generate your own token following the documentation under `docs/acs/test-locally-static-token.md` or
use the token found within Bitwarden (`ACS Fleet* static token`):
```
$ STATIC_TOKEN=<generated value | bitwarden value> AUTH_TYPE=STATIC_TOKEN ./fleetshard-sync
```
