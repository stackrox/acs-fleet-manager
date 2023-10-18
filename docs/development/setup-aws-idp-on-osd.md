# Authenticate ACSCS services on AWS using OSD cluster Identity Provider

## Overview
The goal is to authenticate ACSCS applications (pods) to AWS. The recommended approach is to use STS so that each application assumes an IAM role with policies describing what that application is allowed to do.
Every Openshift cluster (OCP) has a built-in OAuth server that is used to authenticate service accounts within the cluster. OSD is no exception. This document describes how to add an Identity Provider to the AWS IAM configuration to use the OAuth server to authenticate ACSCS pods to AWS.

## Implementation
In scope are the environments that run on OSD clusters, namely _stage_ and _prod_. Engineers may want to set up their own OSD cluster to make experiments. The setup procedure is described [below](#manual-setup-dev).
The main idea is the following: The OAuth server has been added to the AWS IAM configuration as an Identity Provider. Each cluster has its own server, so each must be declared in the AWS IAM configuration. AWS has a limit of 100 identity providers per account [1].
The complication is that server is not reachable from AWS STS, therefore, it couldn't load the public keys to verify a JWT token. The recommended approach is to expose OAuth server's public keys in a **public** S3 bucket [2].

S3 bucket name has the following pattern: `<osdClusterName>-<randomAlphanumeric>-oidc`
- `osdClusterName` is to identify the cluster associated with this bucket
- `randomAlphanumeric` - 32-character random string to ensure uniqueness of the S3 bucket
- `oidc` suffix is chosen to indicate that bucket corresponds to an OAuth server. This suffix also corresponds to the pattern used in the `ccoctl` tool [3].

## Manual setup (dev)
> ❗️This instruction is intended only for OSD clusters. For local clusters, it is recommended to use the static token (AWS_STATIC_TOKEN env variable) for authentication.

1. Install `ccoctl`
   ```shell
   go install github.com/openshift/cloud-credential-operator/cmd/ccoctl@latest
   ```
1. Prepare the environment
    ```shell
    ENVIRONMENT=dev
    AWS_REGION=us-east-1
    GITROOT=$(git rev-parse --show-toplevel)
    CLUSTER_ID="<ID of the cluster>"
    CLUSTER_NAME="<name of the cluster>"
    IDP_NAME="${CLUSTER_NAME}-${CLUSTER_ID}"
    OUTPUT_DIR="${IDP_NAME}"
    cd $GITROOT/tmp
    mkdir $OUTPUT_DIR
    ```
1. Login into the osd cluster
    ```shell
    ocm cluster login $CLUSTER_ID
    # follow instructions to login
    ```
1. Retrieve a public key from the cluster
    ```shell
    oc get configmap --namespace openshift-kube-apiserver bound-sa-token-signing-certs --output json | jq --raw-output '.data["service-account-001.pub"]' > "${OUTPUT_DIR}/serviceaccount-signer.public"
    ```
1. Authenticate in AWS
    ```saml
    make -C $GITROOT $GITROOT/bin/tools_venv
    source $GITROOT/bin/tools_venv/bin/activate
    aws-saml.py
    ```
1. Create an Identity Provider
    ```shell
    ccoctl aws create-identity-provider --output-dir $OUTPUT_DIR --name $IDP_NAME --region $AWS_REGION
    ```
1. Change the cluster authentication
    ```shell
    oc apply -f ./$OUTPUT_DIR/manifests/cluster-authentication-02-config.yaml
    ```
1. Make sure that the cluster authentication has an appropriate url pointing to the S3 bucket
    ```shell
    oc get authentication -o yaml
    ```

## Tear down
```shell
ccoctl aws delete --name $IDP_NAME --region $AWS_REGION
deactivate
```

## Links
1. https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_iam-quotas.html
1. https://github.com/openshift/enhancements/blob/master/enhancements/cloud-integration/aws/aws-pod-identity.md
1. https://github.com/openshift/cloud-credential-operator/blob/master/docs/ccoctl.md
1. https://access.redhat.com/documentation/en-us/openshift_container_platform/4.5/html-single/authentication_and_authorization/index#bound-sa-tokens-about_bound-service-account-tokens
