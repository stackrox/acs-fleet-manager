> NOTE some step in this document might refer to Red Hat internal components
  that you do not have access to

# Populating Configuration
The document describes how to prepare Fleet Manager to be able to start by
populating its configurations.

Follow all subsections to get a bootable Fleet Manager server.

## Interacting with the Fleet Manager API

The Fleet Manager API requires OIDC Bearer tokens, which use the JWT format,
for authentication. The token is passed as an `Authorization` HTTP header value.

To be able to perform requests to the Fleet Manager API, you first need
to get a JWT token from the configured OIDC authentication server, which is
Red Hat SSO (sso.redhat.com) by default. Assuming the default OIDC authentication server is
being used, this can be performed by interacting with the OCM API. This can be
easily done interacting with it through the [ocm cli](https://github.com/openshift-online/ocm-cli/releases)
and retrieve OCM tokens using it. To do so:
1. Login to your desired OCM environment via web console using your Red Hat
   account credentials. For example, for the OCM production environment, go to
   https://cloud.redhat.com and login.
1. Get your OCM offline token by going to https://cloud.redhat.com/openshift/token
1. Login to your desired OCM environment through the OCM CLI by providing the
   OCM offline token and environment information:
   ```
   ocm login --token <ocm-offline-token> --url <ocm-api-url>
   ```
   `<ocm-api-url>` is the URL of the OCM API. Some shorthands can also
   be provided like `production` or `staging`
1. Generate an OCM token by running: `ocm token`. The generated token is the token
   that should be used to perform a request to the Fleet Manager API. For example:
  ```
  curl -H "Authorization: Bearer <result-of-ocm-token-command>" http://127.0.0.1:/8000/api/dinosaurs_mgmt
  ```
  OCM tokens other than the OCM offline token have an expiration time so a
  new one will need to be generated when that happens

There are some additional steps needed if you want to be able to perform
certain actions that have additional requirements. See the
[_User Account & Organization Setup_](getting-credentials-and-accounts.md#user-account--organization-setup) for more information

## Setting up OCM tokens

The Fleet Manager itself requires the use of an OCM token so it can
interact with OCM to perform management of Data Plane clusters.

In order for the Fleet Manager to do so, an OCM offline token should be configured.
To do so, retrieve your OCM offline token and then configure it for Fleet
Manager by running:
```
make ocm/setup OCM_OFFLINE_TOKEN=<your-retrieved-ocm-offline-token>
```

## Setup AWS configuration
Fleet Manager interacts with AWS to provide the following functionalities:
* To be able to create and manage Data Plane clusters in a specific AWS account
  by passing the needed credentials to OpenShift Cluster Management
* To create [AWS's Route53](https://aws.amazon.com/route53/) DNS records in a
  specific AWS account. These records are DNS records that point to some
  routes related to Central instances that are created.
  > NOTE: The domain name used for these records can be configured by setting
    the domain name to be used for Central instances. This can be done
    through the `--central-domain-name` Fleet Manager binary CLI flag
For both functionalities, the same underlying AWS account is used.

In order for the Fleet Manager to be able to start, create the following files:
```
touch secrets/aws.accountid
touch secrets/aws.accesskey
touch secrets/aws.secretaccesskey
touch secrets/aws.route53accesskey
touch secrets/aws.route53secretaccesskey
```

If you need any of those functionalities keep reading. Otherwise, this section
can be skipped.

To accomplish the previously mentioned functionalities Fleet Manager needs to
be configured to interact with the AWS account. To do so, provide existing AWS
IAM user credentials to the control plane by running:
```
AWS_ACCOUNT_ID=<aws-account-id> \
AWS_ACCESS_KEY=<aws-iam-user-access-key> \
AWS_SECRET_ACCESS_KEY=<aws-iam-user-secret-access-key> \
ROUTE53_ACCESS_KEY=<aws-iam-user-for-route-53-access-key> \
ROUTE53_SECRET_ACCESS_KEY=<aws-iam-user-for-route-53-secret-access-key> \
make aws/setup
```
> NOTE: If you are in Red Hat, the following [documentation](./getting-credentials-and-accounts.md#aws)
  might be useful to get the IAM user/s credentials

## Setup RedHat SSO configuration

Our default authentication server is provided by RedHat SSO and we have to configure
Fleet Manager to use it.

In order for the Fleet Manager to be able to start, create the following files:
```
touch secrets/redhatsso-service.clientId
touch secrets/redhatsso-service.clientSecret
```

If you have RedHat SSO service account defined for Fleet Manager,
you can set Fleet Manager to use them by running the following command:
```
 SSO_CLIENT_ID="<redhatsso-client-id>" \
 SSO_CLIENT_SECRET="<redhatsso-client-secret" \
 make redhatsso/setup
```

## Setup the data plane image pull secret
In the Data Plane cluster, the Central Operator and the FleetShard Deployments
might reference container images that are located in authenticated container
image registries.

Fleet Manager can be configured to send this authenticated
container image registry information as a K8s Secret in [`kubernetes.io/.dockerconfigjson` format](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#registry-secret-existing-credentials).

In order for the Fleet Manager to be able to start, create the following file:
```
touch secrets/image-pull.dockerconfigjson
```

If you don't need to make use of this functionality you can skip this section.
Otherwise, keep reading below.

To configure the Fleet Manager with this authenticated registry information so
the previously mentioned Data Plane elements can pull container images from it:
* Base-64 encode your [Docker configuration file](https://docs.docker.com/engine/reference/commandline/cli/#docker-cli-configuration-file-configjson-properties).
* Copy the contents generated from the previous point into the `secrets/image-pull.dockerconfigjson` file

## Setup the Observability stack secrets
See [Obsevability](./observability/README.md) to learn more about Observatorium and the observability stack.
The following command is used to setup the various secrets needed by the Observability stack.

```
make observatorium/setup
```

## Setup a custom TLS certificate for Central Host URLs

When Fleet Manager creates Central instances, it can be configured to
send a custom TLS certificate to associate to each one of the Central instances
host URLs. That custom TLS certificate is sent to the data plane clusters where
those instances are located.

In order for the Fleet Manager to be able to start, create the following files:
```
touch secrets/central-tls.crt
touch secrets/central-tls.key
```

If you need to setup a custom TLS certificate for the Central instances' host
URLs keep reading. Otherwise, this section can be skipped.

To configure Fleet Manager so it sends the custom TLS certificate, provide the
certificate and its corresponding key to the Fleet Manager by running the
following command:
```
CENTRAL_TLS_CERT=<central-tls-cert> \
CENTRAL_TLS_KEY=<central-tls-key> \
make centralcert/setup
```
> NOTE: The certificate domain/s should match the URL endpoint domain if you
  want the certificate to be valid when accessing the endpoint
> NOTE: The expected Certificate and Key values are in PEM format, preserving
  the newlines

Additionally, make sure that the functionality is enabled by setting the
`--enable-central-external-certificate` Fleet Manager binary CLI flag

## Configure Sentry logging
Fleet Manager can be configured to send its logs to the
[Sentry](https://sentry.io/) logging service.

In order for the Fleet Manager to be able to start, create the following files:
```
touch secrets/sentry.key
```

If you want to use Sentry set the Sentry Secret key in the `secrets/sentry.key`
previously created.

Additionally, make sure to set the Sentry URL endpoint and Sentry project when
starting the Fleet Manager server. See [Sentry-related CLI flags in Fleet Manager](./feature-flags.md#sentry)
