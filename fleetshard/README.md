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

See [secret management docs](docs/development/secret-management.md) for more information and tips.

Dev environment is selected by default. After this you may call
```shell
./dev/env/scripts/exec_fleetshard_sync.sh
```
to inject the necessary environment variables to the fleetshard-sync application.

## Authentication types

Fleetshard sync provides different authentication types that can be used when calling the fleet manager's API.

### Red Hat SSO

This is the default authentication type used.
To run fleetshard-sync with RH SSO, use the following command:
```shell
./dev/env/scripts/exec_fleetshard_sync.sh
```

### OCM Refresh token

To run fleetshard-sync with the OCM refresh token, use the following:
```shell
OCM_TOKEN=$(ocm token --refresh) \
AUTH_TYPE=OCM \
./dev/env/scripts/exec_fleetshard_sync.sh
```

### Static token

A static token has been created which is non-expiring. The JWKS certs are by default added to fleet manager.
The token's claims can be viewed under `config/static-token-payload.json`.
You can either generate your own token following the documentation under `docs/acs/test-locally-static-token.md` or
use the token found within Bitwarden (`ACS Fleet* static token`):
```
STATIC_TOKEN=<generated value | bitwarden value> \
AUTH_TYPE=STATIC_TOKEN \
./dev/env/scripts/exec_fleetshard_sync.sh
```

### Manage ACS Operator(s)

Fleetshard-sync service is able to manager installation/update
of ACS Operator based on running and desired ACS Instances versions.
Fleetshard-sync operator ACS Operator management should replace OLM based approach.

#### Rolling out installation/update of ACS Operator:

1. Make sure that OLM ACS Operator subscription is deleted.
OLM uses the subscription resource to subscribe to the latest version of an operator.
OLM reinstalls a new version of the operator even if the operator’s CSV was deleted earlier.
In effect, you must tell OLM that you do not want new versions of the operator to be installed by deleting the ACS Operator subscription
```
kubectl get subscription -n stackrox-operator
kubectl delete subscription <subscription> -n stackrox-operator
```

2. Delete the Operator’s ClusterServiceVersion.
The ClusterServiceVersion contains all the information that OLM needs to manage an operator,
and it effectively represents an operator that is installed on cluster

```
kubectl get clusterserviceversion -n stackrox-operator
kubectl delete clusterserviceversion rhacs-operator.<version> -n stackrox-operator
```

3. Check that there is no running ACS Operator

```
kubectl get pods -n stackrox-operator
NAME                                                              READY   STATUS      RESTARTS      AGE
```

4. Turn on ACS Operator management feature flag

set `FEATURE_FLAG_UPGRADE_OPERATOR_ENABLED` to `true` and redeploy Fleetshard-sync service

5.  Check that the ACS Operator is running again

```
kubectl get pods -n stackrox-operator
NAME                                                              READY   STATUS      RESTARTS       AGE
rhacs-operator-controller-manager-3.74.1-5765676ffc-l9bpp         2/2     Running     0              13s
...
```

5.  Check deployment

```
kubectl get deployments -n stackrox-operator
NAME                                       READY   UP-TO-DATE   AVAILABLE   AGE
rhacs-operator-controller-manager-3.74.1   1/1     1            1           27s
```
