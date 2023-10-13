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
