# How-To setup developer ROSA cluster (step by step copy/paste guide)

### Prerequisites

You will require several commands in order to use simple copy/paste.
1. [rosa](https://console.redhat.com/openshift/downloads) - CLI for Red Hat OpenShift Service on AWS.
1. [aws-saml.py](https://gitlab.corp.redhat.com/compute/aws-automation) - helper tool for authenticating in AWS using SAML
1. `bw` - BitWarden CLI. We need this to get values from BitWarden directly without paste/copy.
1. `oc` - OpenShift cluster CLI tool (similar to kubectl). We need it to deploy resources into the ROSA cluster.
1. `ocm` - OpenShift cluster manager CLI tool. We need it to extend cluster lifetime and create Centrals.
1. `roxctl` - StackRox CLI for managing Central and downloading cluster registration secrets.

### Intro

All commands should be executed in root directory of `stackrox/acs-fleet-manager` project.

### Create development ROSA Cluster with `rosa` CLI

1. Login to OCM staging

    Export name for your cluster. Prefix it with your initials or something similar to avoid name collisions. i.e. `mt-rosa-1307`
    ```shell
    ROSA_CLUSTER_NAME="johndoe-test" # use your name
    ```

    To create development ROSA cluster in OCM staging platform, you should login to staging platform. You should use `rhacs-managed-service-dev` account.
    To retrieve token required to login via `rosa` command:
    ```shell
    rosa login --url staging --use-device-code
    ```
    Please follow the instructions to open the browser and enter the code from the command output. Login there as `rhacs-managed-service-dev`. You can find `rhacs-managed-service-dev` login credentials in BitWarden.
    The `rosa` command is aware of differences and defining `--url staging` is all what is required in order to login to OCM staging platform.

1. Login to AWS dev account
    Run `aws-saml.py` and select dev account (see [secret-management.md](./secret-management.md)).

1. Select the desired AWS region
    ```shell
    export AWS_REGION="us-east-1"
    ```

1. Preflight checks

    Double check selected AWS account, AWS region,  OCM user and OCM staging API
    ```shell
    rosa whoami
    ```

1. Create cluster
    ```shell
    rosa create cluster --cluster-name "${ROSA_CLUSTER_NAME}" --sts --mode auto
    ```
    Follow the interactive instructions.
    If prompted for the IAM account roles, create them
    ```shell
    rosa create account-roles
    ```
    Now, you have to wait for cluster to be provisioned. Check status of cluster creation:
    ```shell
    rosa logs install -c "${ROSA_CLUSTER_NAME}" --watch
    ```

1. Create admin user

    This is required in order to be able to log in to cluster in the UI or with `oc` command.

    ```shell
    rosa create admin -c $ROSA_CLUSTER_NAME
    ```

    The command will output the `oc login` command containing the cluster kube API URL, username and password.
    Please securely store this generated password. If you lose this password you can delete and recreate the cluster admin user.

1. Login to the cluster

   Use the command output from the previous step:
    ```shell
    oc login <cluster_api_url> --username cluster-admin --password <..generated..>
    ```
    If login step fails, it can be the case that previously created auth provider and user are not applied yet on the cluster. You can wait few seconds and try again.

### Deploy ACSCS

```shell
export CLUSTER_TYPE=infra-openshift
make deploy/bootstrap deploy/dev
```
See [setup-test-environment.md](setup-test-environment.md) for more info.

### Install central
1. Prerequisites:
   1. [step](https://github.com/smallstep/cli) CLI
   2. `roxctl`
1. Log in with your personal account to **stage** RH SSO and capture the OAuth token
    ```shell
    OAUTH_TOKEN=$(step oauth --bare \
      --client-id="cloud-services" \
      --provider="https://sso.stage.redhat.com/auth/realms/redhat-external" \
      --scope="openid")
    ```
1. Create Central
    ```shell
    curl -X POST -H "Authorization: Bearer ${OAUTH_TOKEN}" -H "Content-Type: application/json" \
      http://127.0.0.1:8000/api/rhacs/v1/centrals\?async\=true \
      -d '{"name": "rosa-test", "multi_az": true, "cloud_provider": "standalone", "region": "standalone"}'
    ```
1. Capture `id` JSON field from the command output
    ```shell
    CENTRAL_ID=<id from JSON>
    ```
1. Set `CENTRAL_NAMESPACE` environment variable
    ```shell
    CENTRAL_NAMESPACE="rhacs-${CENTRAL_ID}"
    ```
1. Check if new namespace is created and if all pods are up and running
    ```shell
    oc get pods -n "$CENTRAL_NAMESPACE"
    ```
1. Set `CENTRAL_ENDPOINT` environment variable
    ```shell
    CENTRAL_ENDPOINT="$(oc get route managed-central-reencrypt -n $CENTRAL_NAMESPACE -o jsonpath="{.spec.host}"):443"
    ```

### Install sensor to same data plane cluster where central is installed
1. Login to central
    ```shell
    roxctl --endpoint "${CENTRAL_ENDPOINT}" --insecure-skip-tls-verify central login
    ```
1. Generate CRS
    ```shell
    roxctl --endpoint "${CENTRAL_ENDPOINT}" --insecure-skip-tls-verify central crs generate rosa-test-secured-cluster --output /tmp/rosa-test-secured-cluster-crs.yaml  
    ```
1. Install sensor
    ```shell
    oc create ns rhacs-secured-cluster
    oc apply -n rhacs-secured-cluster -f /tmp/rosa-test-secured-cluster-crs.yaml
    oc apply -n rhacs-secured-cluster -f - <<EOF
apiVersion: platform.stackrox.io/v1alpha1
kind: SecuredCluster
metadata:
  name: stackrox-secured-cluster-services
  labels:
    rhacs.redhat.com/selector: dev
spec:
  clusterName: rosa-test
  centralEndpoint: ${CENTRAL_ENDPOINT}
  admissionControl:
    resources:
      limits:
        memory: 150Mi
      requests:
        cpu: 100m
        memory: 150Mi
    replicas: 1
  sensor:
    resources:
      limits:
        memory: 100Mi
      requests:
        cpu: 10m
        memory: 100Mi
  scanner:
    scannerComponent: Disabled
  scannerV4:
    scannerComponent: Disabled
EOF
    ```
1. Check that sensor is up and running
    ```
    oc get pods -n rhacs-secured-cluster
    ```
### Run local front-end (UI project)

The front-end is located in the following repo: https://github.com/RedHatInsights/acs-ui. Clone that repo locally.

1. Prepare `/etc/hosts` file. Add development host to the hosts file. The grep command ensures that entry is added only once.
    ```
    sudo sh -c 'grep -qxF "127.0.0.1 stage.foo.redhat.com" /etc/hosts || echo "127.0.0.1 stage.foo.redhat.com" >> /etc/hosts'
    ```
    **Note:** If you are unsure what the command will do, be free to manually add the entry `127.0.0.1 stage.foo.redhat.com` in the `/etc/hosts` file.
1. Install the UI project.
Execute the following commands in the root directory of the UI project:
    ```
    npm install
    ```
1. Start the UI project. Execute the following commands in the root directory of the UI project:
    ```
    export FLEET_MANAGER_API_ENDPOINT=http://localhost:8000

    npm run start:beta
    ```
    After that, you can open the following URL in your browser: https://stage.foo.redhat.com:1337/beta/application-services/acs

    **Note:** Since staging External RedHat SSO is used for authentication, you may have to create your personal account.

### Extend development ROSA cluster lifetime to 7 days


By default, staging cluster will be up for 2 days. You can extend it to 7 days.

Determine cluster's ID:
```shell
rosa describe cluster -c $ROSA_CLUSTER_NAME
```
Capture the ID value:
```shell
CLUSTER_ID=<value of the ID row>
```

Execute the following command for macOS:
```
echo "{\"expiration_timestamp\":\"$(date -v+7d -u +'%Y-%m-%dT%H:%M:%SZ')\"}" | ocm patch "/api/clusters_mgmt/v1/clusters/${CLUSTER_ID}"
```

Or on Linux:
```
echo "{\"expiration_timestamp\":\"$(date --iso-8601=seconds -d '+7 days')\"}" | ocm patch "/api/clusters_mgmt/v1/clusters/${CLUSTER_ID}"
```

## See also
1. [ROSA quick start guide](https://docs.redhat.com/en/documentation/red_hat_openshift_service_on_aws/4/html/install_clusters/rosa-hcp-quickstart-guide)
