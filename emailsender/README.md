# ACSCS Email Service

The email service which allows sending email notification
from ACS Central tenants without bringing an own SMTP service.

Central tenants call this cluster-internal service by the dedicated notifier integration type [acscsEmail](https://github.com/stackrox/stackrox/tree/master/central/notifiers/acscsemail).

## Quickstart

```sh
make db/setup # if no local db started yet
make run/emailsender # start email sender service

# to be able to run a full integration test login
# to the AWS dev environment before starting the service

# Sample request to send a test email, rawMessage is b64 encoded msg content
curl localhost:8080/api/v1/acscsemail -v -XPOST \
-H "Content-Type: application/json" \
-H "Authorization: Bearer $(ocm token)" \
--data-raw '{"to":["success@simulator.amazonses.com"], "rawMessage":"dGVzdCBtZXNzYWdlIGNvbnRlbnQ="}'
```

If you want to send an actual email to your inbox [verify](https://docs.aws.amazon.com/ses/latest/dg/creating-identities.html#verify-email-addresses-procedure) your email address for use in AWS SES using the AWS Web Console for the Dev Account.

## Rate Limitting

Emailsender has a rate limit per tenant for sending emails (Default: 250). The tenant is identified by the `sub` claim of the token used to call the API.

[TODO] Reference limit documentation in runbooks repository

## Deployment

The emailsender is deployed as part of the acs-fleetshard-sync addon. The helm chart is defined in `dp-terraform/helm/rhacs-terraform/templates/emailsender*.yaml`.

The most important helm values are exposed for configuration through the addon.

For more details refer to the addon [documentation](https://spaces.redhat.com/pages/viewpage.action?spaceKey=StackRox&title=ACS+Fleetshard+Addon).

## API Authentication

The emailsender exposes an HTTP/S API. There are 2 authentication methods to call that API.

- OCM Tokens (default for dev setup)
- Kubernetes Service Accounts (used in ACSCS environment)

### OCM Token Authentication

The file `config/emailsender-authz.yaml` configures the OCM token authentication. On a local machine emailsender will read this file and allow you to call the API using your personal OCM token, provided you are in the organizations listed in `allowed_org_ids`.

To use this authentication type when emailsender is deployed to a K8s cluster you have to create a ConfigMap like this and mount it to the pod. The filepath is configurable by the environment variable `AUTH_CONFIG_FILE`

### Kubernetes Service Account Authentication

The preferred method for authentication when running emailsender in a K8s cluster is to the Kubernetes Service Account. To activate this set the env variable `AUTH_CONFIG_FROM_KUBERNETES=true`.

Emailsender will read the OIDC config used to issue Kubernetes service account token from the Kubernetes API server.

In this authentication mode all service account token with a sub claim matching the regexp `system:serviceaccount:rhacs-[a-z0-9]*:central`.
