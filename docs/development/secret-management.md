# Secret Management

## Overview / Tools
Application Secrets are stored in AWS Secrets Manager.
The following tools are used to integrate with Secrets Manager:
- [chamber](https://github.com/segmentio/chamber) - CLI for managing secrets
- [aws-saml.py](https://gitlab.corp.redhat.com/compute/aws-automation) - helper tool for authenticating in AWS using SAML

The main usage is to load the secrets as environment variables for deploying a service.
Secrets are divided to subgroups per each service. The following services are currently exist:

**Application specific:**
- fleet-manager
- fleetshard-sync
- logging
- observability

**Cluster specific:**
- acs-stage-dp-01
- acs-prod-dp-01

## Instructions
- `AWS_AUTH_HELPER` environment variable selects the appropriate authentication method within the deployment scripts. Possible options are:
  - `aws-saml`
  - `none` (default)
- Depending on the environment, the following choices are set:

    | Source | Target         | AWS_AUTH_HELPER |
    |--------|----------------|-----------------|
    | local  | dev,stage,prod | aws-saml        |
    | CI/CD  | stage,prod     | none            |

- For SAML authentication, you must have access to the [`aws-automation` git repository](https://gitlab.corp.redhat.com/compute/aws-automation) for the script to be able to download the tool (VPN is required).
- Dependent scripts source the [helper script](./../../scripts/lib/external_config.sh) with `chamber` command wrapper;
- With this script, the tools are automatically installed from the appropriate `Makefile` targets;
- It is also recommended to install the tools in the local bin folder so that you can easily use `chamber` and `aws-saml.py` from the command line;

## Tips / Examples

### Setup Kerberos
#### Use macOS Keychain for Kerberos
macOS users may leverage Keychain to seamlessly refresh the Kerberos ticket. Execute this command once:
```shell
kinit --keychain
```
Subsequent `kinit` invocations will not require a password.
### Write secret
```shell
chamber write -b secretsmanager <service> <KEY> -
<value>
^D # end-of-stdin
```
for example:
```shell
chamber write -b secretsmanager fleetshard-sync RHSSO_SERVICE_ACCOUNT_CLIENT_ID -
changeme
^D
```

### Print environment
```shell
chamber env -b secretsmanager fleetshard-sync
```

## Troubleshooting
#### kinit: krb5_get_init_creds: Error from KDC: CLIENT_NOT_FOUND
Check if your OS user matches the company User ID. If not, then specify it explicitly:
```shell
kinit bob
```
