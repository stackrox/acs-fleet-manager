# Data Plane OSD cluster Options
## Using an existing OSD Cluster

Any OSD cluster can be used by the service, it does not have to be created with the service itself. If you already have an existing OSD
cluster, you will need to register it in the database so that it can be used by the service for incoming Dinosaur requests.  The cluster must have been created with multizone availability.

Get the ID of your cluster (e.g. `1h95qckof3s31h3622d35d5eoqh5vtuq`) using the CLI
        - Run `ocm list clusters`
        - The ID should be displayed under the ID column

>NOTE: By default the auto scaling feature is disabled in development mode. You can enable it by passing `--dataplane-cluster-scaling-type=auto` when starting a built binary. Or changing the default value in [default development environment flags file](../internal/dinosaur/pkg/environments/development.go) when using `make run` to start the API.

## Using an existing OSD cluster with manual scaling enabled

You can manually add the cluster in the [dataplane-cluster-configuration.yaml](../config/dataplane-cluster-configuration.yaml) file.

A content of the file is:

- A list of clusters for fleet manager
- All clusters are created with `cluster_provisioning` status if they are not already in fleet manager
- All clusters in fleet manager DB already but are missing in the list will be marked as `deprovisioning` and will later be deleted.
- This list is ordered, any new cluster should be appended at the end.

From the cluster ID taken from step (1), use `ocm` CLI to get cluster details i.e `name`, `id` which is cluster_id, `multi_az`, `cloud_provider` and `region`.
```shell
ocm get /api/clusters_mgmt/v1/clusters/$ID | jq ' .name, .id, .multi_az, .cloud_provider.id, .region.id '

"cluster-name"
"1jp6kdr7k0sjbe5adck2prjur8f39378" # or a value matching your cluster ID
true
"aws" # or any cloud provider
"us-east-1" # or any region
```

From the above command, the resulting [dataplane-cluster-configuration.yaml](../config/dataplane-cluster-configuration.yaml) file content will
```yaml
clusters:
  - name: cluster-name
    cluster_id: 1jp6kdr7k0sjbe5adck2prjur8f39378
    cloud_provider: aws
    region: us-east-1
    multi_az: true
    schedulable: true # change this to false if you do not want the cluster to be schedulable
    dinosaur_instance_limit: 2 # change this to match any value of configuration
    supported_instance_type: "standard,eval" # could be "eval", "standard" or both i.e "standard,eval" or "eval,standard". Defaults to "standard,eval" if not set
```
### Connecting to a standalone cluster

fleet-manager allows provisioning of dinosaurs in an already preexisting standalone dataplane cluster. To do so, add the standalone cluster in the [dataplane-cluster-configuration.yaml](../config/dataplane-cluster-configuration.yaml) giving the:
 - `name` of the kubeconfig context to use. This option is required and it has to be an existing name of a context in kubeconfig
 - `provider_type` must be set to `standalone`. This is required to indicate that we are using a standalone
 - `cluster_dns` This will be used to build dinosaur host url and to communicate with clusters e.g `apps.example.dns.com`. This option is required.
 - `cloud_provider` the cloud provider where the standalone cluster is provisioned in
 - `region` the cloud region where the standalone cluster is provisioned
 - ... rest of the options

> NOTE: `kubeconfig` path can be configured via the `--kubeconfig` CLI flag. Otherwise is defaults to `$HOME/.kube/config`

> NOTE: [OLM](https://github.com/operator-framework/operator-lifecycle-manager#installation) in the destination standalone cluster/s is a prerequisite to be able to install dinosaur operator and fleetshard operators

## Configuring OSD Cluster Creation and AutoScaling

To configure auto scaling, use the `--dataplane-cluster-scaling-type=auto`.
Once auto scaling is enabled this will activate the scaling up/down of compute nodes for existing clusters, dynamic creation and deletion of OSD dataplane clusters as explained in the [dynamic scaling architecture documentation](./architecture/data-plane-osd-cluster-dynamic-scaling.md)

## Registering an existing cluster in the Database

>NOTE: This should only be done if auto scaling is enabled. If manual scaling is enabled, please follow the guide for [using an existing cluster with manual scaling](#using-an-existing-osd-cluster-with-manual-scaling-enabled) instead.

1. Register the cluster to the service
    - Run the following command to generate an **INSERT** command:
      ```
      make db/generate/insert/cluster CLUSTER_ID=<your-cluster-id>
      ```
    - Run the command generated above in your local database:
        - Login to the local database using `make db/login`
        - Execute SQL command from previous steps (fleet-manager will populate any blank values when it reconciles)
        - Ensure that the **clusters** table is available.
            - Create the binary by running `make binary`
            - Run `./fleet-manager migrate`
        - Once the table is available, the generated **INSERT** command can now be run.

2. Ensure the cluster is ready to be used for incoming Dinosaur requests.
    - Take note of the status of the cluster, `cluster_provisioned`, when you registered it to the database in step 2. This means that the cluster has been successfully provisioned but still have remaining resources to set up (i.e. Dinosaur operator installation).
    - Run the service using `make run` and let it reconcile resources required in order to make the cluster ready to be used by Dinosaur requests.
    - Once done, the cluster status in your database should have changed to `ready`. This means that the service can now assign this cluster to any incoming Dinosaur requests so that the service can process them.
