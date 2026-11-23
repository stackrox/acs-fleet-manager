## Debugging

## Links

Job Definition: https://github.com/openshift/release/tree/master/ci-operator/jobs/stackrox/acs-fleet-manager
Config Definition: https://github.com/openshift/release/tree/master/ci-operator/config/stackrox/acs-fleet-manager

### Access Job Cluster and Real Time Logs

- To access the OpenShift UI to view the logs directly search for something like `Using namespace https://console-openshift-console.apps.build04.34d2.p2.openshiftapps.com/k8s/cluster/projects/ci-op-0b6vixvb `.
- Access OpenShift UI, open `Administartor` overview on the top left.
- View the `Environment`, copy the `KUBECONFIG` path, open the Pod's `Terminal` view in the UI and run `cat <KUBECONFIG_PATH>`
- Copy KUBECONFIG content and create the KUBECONFIG locally
- Run `export KUBECONFIG=/local/path/to/kubeconfig`

### Check Fleet-Manager logs and build logs

The build log contains the test output, a cluster summary (pods, GitOps apps, warning events)
and the full stern log of the ACSCS namespace (fleet-manager, fleetshard-sync, emailsender,
RHACS operator), e.g. `== BEGIN LOG namespace-rhacs.txt ==`.

Everything else is written to `$ARTIFACT_DIR` and uploaded with the job artifacts under
`artifacts/e2e/test/artifacts/`:
```
junit_e2e.xml, e2e-report.json  test reports (failed tests are shown on the Prow job page)
pod-logs/                       stern logs per namespace (continuous, across pod restarts)
acscs/                          describe pods, full logs per pod (fleet-manager, fleetshard-sync, ...)
operator/                       pods, subscriptions, installplans
external-secrets/               cluster secret stores, external secrets, ESO logs
tenants/<namespace>/            Central CR, deployments, pods, events, Argo CD app, full logs per pod
gitops/                         Argo CD applications
```

Locally, `.openshift-ci/tests/e2e.sh` writes to a fresh `mktemp -d` directory per run unless
`ARTIFACT_DIR` is set. The directory is printed at the start and end of the run.
