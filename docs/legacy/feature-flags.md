# [Deprecated] Note

To introduce new feature flags please use package `pkg/features/` and add your flag to `list.go`.

# Feature Flags

This lists the feature flags and their sub-configurations to enable/disable and configure features of the Fleet Manager. This set of features can be seen below.

  - [Feature Flags](#feature-flags)
  - [Access Control](#access-control)
  - [Database](#database)
  - [Health Check Server](#health-check-server)
  - [Central](#central)
  - [IAM](#iam)
  - [Metrics Server](#metrics-server)
  - [Observability](#observability)
  - [OpenShift Cluster Manager](#openshift-cluster-manager)
  - [Dataplane Cluster Management](#dataplane-cluster-management)
  - [Sentry](#sentry)
  - [Server](#server)

## Access Control
> For more information on access control for Fleet Manager, see this [documentation](./access-control.md).

- **enable-deny-list**: Enables access control for denied users.
    - `deny-list-config-file` [Required]: The path to the file containing the list of users that should be denied access to the service. (default: `'config/deny-list-configuration.yaml'`, example: [deny-list-configuration.yaml](../config/deny-list-configuration.yaml)).

## Database
- **enable-db-debug**: Enables Postgres debug logging.

## Health Check Server
- **enable-health-check-https**: Enable HTTPS for health check server.
    - `https-cert-file` [Required]: The path to the file containing the TLS certificate.
    - `https-key-file` [Required]: The path to the file containing the TLS private key.

## Central
- **enable-deletion-of-expired-central**: Enables deletion of eval Central instances when its life span has expired.
    - `central-lifespan` [Optional]: The desired lifespan of a Central instance in hour(s) (default: `48`).
- **enable-central-external-certificate**: Enables custom Central TLS certificate.
    - `central-tls-cert-file` [Required]: The path to the file containing the Central TLS certificate (default: `'secrets/central-tls.crt'`).
    - `central-tls-key-file` [Required]: The path to the file containing the Central TLS private key (default: `'secrets/central-tls.key'`).
- **enable-evaluator-instance**: Enable the creation of one central evaluator instances per user

- **central-idp-***: A collection of flags describing _static_ auth config for Central.
  If set, every Central will have the **same** IdP config which is likely not what you
  want for production. If not set, the IdP API will be queried for dynamic configuration.
    - **central-idp-client-id**: OIDC client_id to pass to Central's auth config to set
      up its IdP integration.
    - **central-idp-client-secret-file**: File containing OIDC client_secret to pass to
      Central's auth config to set up its IdP integration.
    - **central-idp-issuer**: OIDC issuer URL to pass to Central's auth config to set up
      its IdP integration.

- **quota-type**: Sets the quota service to be used for access control when requesting Central instances (options: `ams` or `quota-management-list`, default: `quota-management-list`).
    > For more information on the quota service implementation, see the [quota service architecture](./architecture/quota-service-implementation) architecture documentation.
    - If this is set to `quota-management-list`, quotas will be managed via the quota management list configuration.
        > See [quota control](./quota-management-list-configuration.md) documentation for more information about the quota management list.
        - `enable-instance-limit-control` [Required]: Enables enforcement of limits on how much Central instances a user can create (default: `false`).

            If enabled, the maximum instances a user can create can be specified in one of the following ways:
            - `quota-management-list-config-file` [Optional]: Allows setting of Central instance limit per organisation
              via _registered_users_per_organisation_ or per service account via _registered_service_accounts_
              (default: `'config/quota-management-list-configuration.yaml'`,
              example: [quota-management-list-configuration.yaml](../config/quota-management-list-configuration.yaml)).
            - `max-allowed-instances` [Optional]: The default maximum Central instance limit a user can create (default: `1`).

            > See the [max allowed instances](./access-control.md#max-allowed-instances) section for more information about setting Central instance limits for users.
    - If this is set to `ams`, quotas will be managed via OCM's accounts management service (AMS).

## IAM
- **redhat-sso-client-id-file** [Required]: The path to the file containing a RedHat SSO account client ID that has access to the ACS service accounts (default: `'secrets/redhatsso-service.clientId'`).
- **redhat-sso-client-secret-file** [Required]: The path to the file containing a RedHat SSO account client secret that has access to the ACS service accounts (default: `'secrets/redhatsso-service.clientSecret'`).

## Metrics Server
- **enable-metrics-https**: Enables HTTPS for the metrics server.
    - `https-cert-file` [Required]: The path to the file containing the TLS certificate.
    - `https-key-file` [Required]: The path to the file containing the TLS private key.

## Observability
- **enable-observatorium-mock**: Enables use of a mock Observatorium client.
    - `observatorium-timeout` [Optional]: Timeout to be used for Observatorium requests (default: `240s`).
- **observatorium-debug**: Enables Observatorium debug logging.
- **observatorium-ignore-ssl**: Disables Observatorium TLS verification.

### Red Hat SSO Authentication
- The '[Required]' in the following denotes that these flags are required to use Red Hat SSO Authentication with the service.
    - `observability-red-hat-sso-auth-server-url`[Required]: Red Hat SSO authentication server URL (default: `https://sso.redhat.com/auth`).
    - `observability-red-hat-sso-realm`[Required]: Red Hat SSO realm (default: `redhat-external`).
    - `observability-red-hat-sso-token-refresher-url`[Required]: Red Hat SSO token refresher URL (default: `www.test.com`).
    - `observability-red-hat-sso-observatorium-gateway`[Required]: Red Hat SSO observatorium gateway (default: `https://observatorium-mst.api.stage.openshift.com`).
    - `observability-red-hat-sso-tenant`[Required]: Red Hat SSO tenant (default: `managedCentral`).
    - `observability-red-hat-sso-logs-client-id-file`[Required]: The path to the file containing the client
    ID for the logs service account for use with Red Hat SSO.
    - `observability-red-hat-sso-logs-secret-file`[Required]: The path to the file containing the client
    secret for the logs service account for use with Red Hat SSO.
    - `observability-red-hat-sso-metrics-client-id-file`[Required]: The path to the file containing the client
    ID for the metrics service account for use with Red Hat SSO.
    - `observability-red-hat-sso-metrics-secret-file`[Required]: The path to the file containing the client
    secret for the metrics service account for use with Red Hat SSO.

## OpenShift Cluster Manager
- **enable-ocm-mock**: Enables use of a mock OCM client.
    - `ocm-mock-mode` [Optional]: Sets the ocm client mock type (default: `stub-server`).
- **ocm-debug**: Enables OpenShift Cluster Manager (OCM) debug logging.

## Dataplane Cluster Management
- **enable-ready-dataplane-clusters-reconcile**: Enables reconciliation of data plane clusters in a `Ready` state.
- **dataplane-cluster-scaling-type**: Sets the behaviour of how the service manages and scales OSD clusters (options: `manual`, `auto` or `none`).
    > For more information on the different dataplane cluster scaling types and their behaviour, see the [dataplane osd cluster options](./data-plane-osd-cluster-options.md) documentation.

    - If this is set to `manual`, the following configuration must be specified:
        - `dataplane-cluster-config-file` [Required]: The path to the file that contains a list of data plane clusters and their details for the service to manage (default: `'config/dataplane-cluster-configuration.yaml'`, example: [dataplane-cluster-configuration.yaml](../config/dataplane-cluster-configuration.yaml)).
    - If this is set to `auto`, the following configurations can be specified:
        - `providers-config-file` [Required]: The path to the file containing a list of supported cloud providers that the service can provision dataplane clusters to (default: `'config/provider-configuration.yaml'`, example: [provider-configuration.yaml](../config/provider-configuration.yaml)).
        - `cluster-compute-machine-type` [Optional]: The compute machine type to be used for provisioning a new dataplane cluster (default: `m5.2xlarge`).
        - `cluster-openshift-version` [Optional]: The OpenShift version to be installed on the dataplane cluster (default: `""`, empty string indicates that the latest stable version will be used).

## Sentry
- **enable-sentry**: Enables Sentry error reporting.
    - `sentry-key-file` [Required]: The path to the file containing the Sentry key (default: `'secrets/sentry.key'`).
    - `sentry-project` [Required]: The Sentry project ID.
    - `sentry-url` [Required]: The base URL of the Sentry instance.
    - `enable-sentry-debug` [Optional]: Enables Sentry debug logging (default: `false`).
    - `sentry-timeout` [Optional]: The timeout duration for requests to Sentry.

## Server
- **enable-https**: Enables HTTPS for the Fleet Manager server.
    - `https-cert-file` [Required]: The path to the file containing the TLS certificate.
    - `https-key-file` [Required]: The path to the file containing the TLS private key.
- **enable-terms-acceptance**: Enables terms acceptance verification.
