# How-To setup OSD cluster Identity Provider (IdP)

## Pre-reqs

1. `ocm` installed
2. Secrets `oidc_client_id` and `oidc_client_secret` set in AWS secrets manager in `us-east-1`.
3. Parameter `oidc_user_list` set by [terraform](https://github.com/stackrox/acs-fleet-manager-aws-config) in `us-east-1`.

Additionally, you will require access to the environment specific AWS account.

## Creating the IdPs

For each OSD cluster, you can create IdPs that will allow login to the OpenShift Console and map your user to a specific group within the cluster, providing i.e. administrative rights.

The following IdPs will be created:
- OIDC IdP using auth.redhat.com as backend.

Before executing the script that manages the IdP setup, you have to ensure you are logged in with OCM.
Based on the environment, you have to choose between `rhacs-managed-service-integration`, `rhacs-managed-service-stage` or `rhacs-managed-service-prod` account.

Afterwards, you can call the script and adjust the parameters based on your needs:
```shell
AWS_REGION=us-east-1 AWS_SAML_ROLE=<aws_account_id>-poweruser ./dp-terraform/osd-cluster-idp-setup.sh "integration|stage|prod" "cluster-name"
```

The script will handle the following:
1. Fetch required parameters from AWS Parameter Store and credentials from AWS Secrets Manager. The first time it runs, it will ask for AWS credentials.
2. Create the OIDC IdP for the cluster.
3. Create the user <-> group mapping for cluster-admins.

Afterwards, you should see the list of users and their group mapping within the console.openshift.com and when
logging in the OSD cluster you should see the option to login via `OIDC`.

**Note: The sync from console.openshift.com and your OSD cluster may take some time for your IdP to show up when trying to log in.**

The script also creates a robot service account and related resources, for use by continuous deployment.

## Cleanup

For the cleanup, there's currently no script offered.

There's two options for deleting the user mappings from console.openshift.com:
- deleting manually within the UI
- calling the following command: `ocm delete user --group=cluster-admins <user id>`

Additionally, you will have to clear the users within the OSD cluster.

You can do so by running the following:
```shell
# Login to the cluster. This will automatically set the correct context for oc.
ocm cluster login <cluster name> --token
oc login --token <copied token> <cluster URL>

# List the identities that have been created. An identity will be created the first time
# a user logins via an IdP
oc get identity

# Delete all identities
oc delete identity <identities>

# Ensure the users are also cleaned up
oc get users

# Delete existing users
oc delete users <user ids>
```

## Troubleshooting

### Authentication error

In case you are receiving an "authentication error" when logging in, here are some steps to further investigate the issue:
```shell
ocm cluster login <cluster name> --token
oc login --token <copied token> <cluster URL>

# Get the authentication operator pods
oc get pods -n openshift-authentication

# Check the logs of the pods (should be 3 replicas) to find an issue
oc logs -n openshift-authentication <pod>
```

The following log statements have been observed and there's a remediation available:
**Please add additional findings, if you have them, in this list to help others!**

`errorpage.go:28] AuthenticationError: users.user.openshift.io <user id> not found`:
Description: This error occurs when the user is deleted within openshift, but the identity is still existing.  
Remediation: Delete the identity of the user. You can do this by running the following:
```shell
# Retrieve all identities
oc get identity
# Use the identity that is related to the user ID
oc delete identity <identity name>
```
