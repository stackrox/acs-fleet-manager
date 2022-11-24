# Secret Management

## Overview / Tools
Application Secrets are stored in AWS Parameter Store.
The following tools are used to integrate with Parameter Store:
- [chamber](https://github.com/segmentio/chamber) - CLI for managing secrets
- [aws-vault](https://github.com/99designs/aws-vault) - supplementary tool to store AWS credentials in the secure local storage
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
  - `aws-vault`
  - `aws-saml`
  - `none` (default)
- Depending on the environment, the following choices are set:

    | Source | Target      | AWS_AUTH_HELPER |
    |--------|-------------|-----------------|
    | local  | dev         | aws-vault       |
    | local  | stage,prod  | aws-saml        |
    | CI/CD  | stage,prod  | none            |

- For SAML authentication, you must have access to the [`aws-automation` git repository](https://gitlab.corp.redhat.com/compute/aws-automation) for the script to be able to download the tool (VPN is required).
- Dependent scripts source the [helper script](./../../scripts/lib/external_config.sh) with `chamber` command wrapper;
- With this script, the tools are automatically installed from the appropriate `Makefile` targets;
- It is also recommended to install the tools in the local bin folder so that you can easily use `chamber`, `aws-saml.py` and `aws-vault` from the command line;

## Tips / Examples
### Useful environment aliases
```shell
alias chamberdev='aws-vault exec dev -- chamber'
alias chamberstage='aws-vault exec stage -- chamber'
alias chamberprod='aws-vault exec prod -- chamber'
```
Without the alias you have to load the session token manually or always add `aws-vault exec` in the beginning.

### Write secret
```shell
chamber write <service> <KEY> -
<value>
^D # end-of-stdin
```
for example:
```shell
chamber write fleetshard-sync RHSSO_SERVICE_ACCOUNT_CLIENT_ID -
changeme
^D
```

### Print environment
```shell
chamber env fleetshard-sync
```
