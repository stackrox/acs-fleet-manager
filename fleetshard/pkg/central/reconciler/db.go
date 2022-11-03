package reconciler

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DBCreatingStatus  = "creating"
	DBAvailableStatus = "available"

	AWSRegion       = "us-east-1" // TODO: this should not be hardcoded
	DBEngine        = "aurora-postgresql"
	DBEngineVersion = "13.7"
	DBInstanceClass = "db.serverless"
	DBUser          = "rhacs_master"
)

func ensureDBProvisioned(ctx context.Context, client ctrlClient.Client, remoteCentralNamespace string) (string, error) {
	rdsClient, err := newRdsClient()
	if err != nil {
		return "", fmt.Errorf("unable to create RDS client, %v", err)
	}

	clusterId := remoteCentralNamespace + "-db-cluster"
	instanceId := remoteCentralNamespace + "-db-instance"

	err = ensureDBClusterCreated(ctx, client, rdsClient, clusterId, remoteCentralNamespace)
	if err != nil {
		return "", fmt.Errorf("ensuring DB cluster %s exists", clusterId)
	}

	err = ensureDBInstanceCreated(rdsClient, instanceId, clusterId)
	if err != nil {
		return "", fmt.Errorf("ensuring DB instance %s exists in cluster %s", instanceId, clusterId)
	}

	return waitForInstanceToBeAvailable(rdsClient, instanceId, clusterId)
}

func ensureDBDeprovisioned(remoteCentralNamespace string) (bool, error) {
	rdsClient, err := newRdsClient()
	if err != nil {
		return false, fmt.Errorf("unable to create RDS client, %v", err)
	}

	clusterId := remoteCentralNamespace + "-db-cluster"
	instanceId := remoteCentralNamespace + "-db-instance"

	//TODO: do not skip taking a final DB snapshot
	if instanceExists(rdsClient, instanceId) {
		//TODO: don't delete if state is "deleting"
		_, err = rdsClient.DeleteDBInstance(newDeleteCentralDBInstanceInput(instanceId, true))
		if err != nil {
			return false, fmt.Errorf("deleting DB instance: %v", err)
		}
	}

	if clusterExists(rdsClient, clusterId) {
		//TODO: don't delete if state is "deleting"
		_, err = rdsClient.DeleteDBCluster(newDeleteCentralDBClusterInput(clusterId, true))
		if err != nil {
			return false, fmt.Errorf("deleting DB cluster: %v", err)
		}
	}

	return true, nil
}

func newRdsClient() (*rds.RDS, error) {
	cfg := &aws.Config{
		Region: aws.String(AWSRegion),
	}

	sess, err := session.NewSession(cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create session, %v", err)
	}

	return rds.New(sess), nil
}

func ensureDBClusterCreated(ctx context.Context, client ctrlClient.Client, rdsClient *rds.RDS, clusterId string, remoteCentralNamespace string) error {
	if !clusterExists(rdsClient, clusterId) {
		// cluster does not exist, create it
		glog.Infof("Provisioning RDS database cluster.")
		dbPassword, err := getDBPassword(ctx, client, remoteCentralNamespace)
		_, err = rdsClient.CreateDBCluster(newCreateCentralDBClusterInput(clusterId, dbPassword))
		if err != nil {
			return fmt.Errorf("creating DB cluster: %v", err)
		}
	}

	return nil
}

func ensureDBInstanceCreated(rdsClient *rds.RDS, instanceId string, clusterId string) error {
	if !instanceExists(rdsClient, instanceId) {
		// instance does not exist, create it
		glog.Infof("Provisioning RDS database instance.")
		_, err := rdsClient.CreateDBInstance(newCreateCentralDBInstanceInput(clusterId, instanceId))
		if err != nil {
			// TODO: delete cluster
			return fmt.Errorf("creating DB instance: %v", err)
		}
	}

	return nil
}

func clusterExists(rdsClient *rds.RDS, clusterId string) bool {
	dbClusterQuery := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(clusterId),
	}

	_, err := rdsClient.DescribeDBClusters(dbClusterQuery)
	if err != nil {
		return false
	}

	return true
}

func instanceExists(rdsClient *rds.RDS, instanceId string) bool {
	dbInstanceQuery := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(instanceId),
		//TODO: add cluster Filter
	}

	_, err := rdsClient.DescribeDBInstances(dbInstanceQuery)
	if err != nil {
		return false
	}

	return true
}

func waitForInstanceToBeAvailable(rdsClient *rds.RDS, instanceId string, clusterId string) (string, error) {
	dbInstanceQuery := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(instanceId),
		//TODO: add cluster Filter
	}

	dbClusterQuery := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(clusterId),
	}

	for {
		result, err := rdsClient.DescribeDBInstances(dbInstanceQuery)
		if err != nil {
			return "", fmt.Errorf("retrieving DB instance state: %v", err)
		}

		if len(result.DBInstances) != 1 {
			return "", fmt.Errorf("unexpected number of DB instances: %v", err)
		}

		dbInstanceStatus := *result.DBInstances[0].DBInstanceStatus
		if dbInstanceStatus == DBAvailableStatus {

			clusterResult, err := rdsClient.DescribeDBClusters(dbClusterQuery)
			if err != nil {
				return "", fmt.Errorf("retrieving DB cluster description: %v", err)
			}

			connectionString := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=require",
				*clusterResult.DBClusters[0].Endpoint, 5432, DBUser, "postgres")

			return connectionString, nil
		} else {
			// TODO: creating is not the only valid status
			if dbInstanceStatus != DBCreatingStatus {
				//TODO: cleanup
				return "", fmt.Errorf("unexpected instance status: %s", dbInstanceStatus)
			}
			time.Sleep(10 * time.Second)
		}
	}
}

func getDBPassword(ctx context.Context, client ctrlClient.Client, remoteCentralNamespace string) (string, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: centralDbSecretName,
		},
	}
	err := client.Get(ctx, ctrlClient.ObjectKey{Namespace: remoteCentralNamespace, Name: centralDbSecretName}, secret)
	if err != nil {
		return "", fmt.Errorf("getting Central DB password from secret: %v", err)
	}

	if dbPassword, ok := secret.Data["password"]; ok {
		return string(dbPassword), nil
	}

	return "", fmt.Errorf("central DB secret does not contain password field: %v", err)
}

func newCreateCentralDBClusterInput(clusterId string, dbPassword string) *rds.CreateDBClusterInput {
	return &rds.CreateDBClusterInput{
		DBClusterIdentifier: aws.String(clusterId),
		Engine:              aws.String(DBEngine),
		EngineVersion:       aws.String(DBEngineVersion),
		MasterUsername:      aws.String(DBUser),
		MasterUserPassword:  aws.String(dbPassword), // TODO: generate random password
		// TODO: the security group needs to be created during the data plane terraforming, and made known to
		// fleet-manager via a configuration parameter. I created this one in the AWS Console.
		VpcSecurityGroupIds: aws.StringSlice([]string{"sg-04dcc23a03646041c"}),
		ServerlessV2ScalingConfiguration: &rds.ServerlessV2ScalingConfiguration{
			MinCapacity: aws.Float64(0.5),
			MaxCapacity: aws.Float64(16),
		},
		BackupRetentionPeriod: aws.Int64(30),

		// TODO: The following are some extra parameters to consider
		// AvailabilityZones: // TODO: should we have multiple AZs?
		// DeletionProtection: // TODO: see if this useful
		// EnableCloudwatchLogsExports // TODO: enable
		// StorageEncrypted // TODO: enable
		// Tags // TODO: e.g. could add a tag that allows us to identify the associated Central
		// PreferredBackupWindow
		// PreferredMaintenanceWindow
	}
}

func newCreateCentralDBInstanceInput(clusterId, instanceId string) *rds.CreateDBInstanceInput {
	return &rds.CreateDBInstanceInput{
		DBInstanceClass:      aws.String(DBInstanceClass),
		DBClusterIdentifier:  aws.String(clusterId),
		DBInstanceIdentifier: aws.String(instanceId),
		Engine:               aws.String(DBEngine),
		PubliclyAccessible:   aws.Bool(true), // TODO: should be false
	}
}

func newDeleteCentralDBInstanceInput(instanceId string, skipFinalSnapshot bool) *rds.DeleteDBInstanceInput {
	return &rds.DeleteDBInstanceInput{
		DBInstanceIdentifier: aws.String(instanceId),
		SkipFinalSnapshot:    aws.Bool(skipFinalSnapshot),
	}
}

func newDeleteCentralDBClusterInput(clusterId string, skipFinalSnapshot bool) *rds.DeleteDBClusterInput {
	return &rds.DeleteDBClusterInput{
		DBClusterIdentifier: aws.String(clusterId),
		SkipFinalSnapshot:   aws.Bool(skipFinalSnapshot),
	}
}
