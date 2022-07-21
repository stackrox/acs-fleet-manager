# Running a Local Observatorium Token Refresher
> NOTE: This is only required if your Observatorium instance is authenticated using sso.redhat.com.

Run the following make target:
```
make observatorium/token-refresher/setup CLIENT_ID=<client-id> CLIENT_SECRET=<client-secret> [OPTIONAL PARAMETERS]
```

**Required Parameters**:
- CLIENT_ID: The client id of a service account that has, at least, permissions to read metrics.
- ClIENT_SECRET: The client secret of a service account that has, at least, permissions to read metrics.

**Optional Parameters**:
- PORT: Port for running the token refresher on. Defaults to `8085`
- IMAGE_TAG: Image tag of the [token-refresher image](https://quay.io/repository/rhoas/mk-token-refresher?tab=tags). Defaults to `latest`
- ISSUER_URL: URL of your auth issuer. Defaults to `https://sso.redhat.com/auth/realms/redhat-external`
- OBSERVATORIUM_URL: URL of your Observatorium instance. Defaults to `https://observatorium-mst.api.stage.openshift.com/api/metrics/v1/manageddinosaur`

>NOTE: Here the tenant used is a dummy one `manageddinosaur`. Once you have your application setup, you will need to have your own tenant created in the Red Hat obsevatorium instance. Optionally, you can deploy a local observatorium instance by following the [observatorium guide](https://github.com/observatorium/observatorium/blob/main/docs/usage/getting-started.md)
