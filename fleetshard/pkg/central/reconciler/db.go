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
	dbAvailableStatus = "available"

	awsRegion       = "us-east-1" // TODO: this should not be hardcoded
	dbEngine        = "aurora-postgresql"
	dbEngineVersion = "13.7"
	dbInstanceClass = "db.serverless"
	dbUser          = "rhacs_master"
)

func ensureDBProvisioned(ctx context.Context, client ctrlClient.Client, remoteCentralNamespace string) (string, error) {
	rdsClient, err := newRdsClient()
	if err != nil {
		return "", fmt.Errorf("unable to create RDS client, %v", err)
	}

	clusterID := remoteCentralNamespace + "-db-cluster"
	instanceID := remoteCentralNamespace + "-db-instance"

	err = ensureDBClusterCreated(ctx, client, rdsClient, clusterID, remoteCentralNamespace)
	if err != nil {
		return "", fmt.Errorf("ensuring DB cluster %s exists", clusterID)
	}

	err = ensureDBInstanceCreated(rdsClient, instanceID, clusterID)
	if err != nil {
		return "", fmt.Errorf("ensuring DB instance %s exists in cluster %s", instanceID, clusterID)
	}

	return waitForInstanceToBeAvailable(rdsClient, instanceID, clusterID)
}

func ensureDBDeprovisioned(remoteCentralNamespace string) (bool, error) {
	rdsClient, err := newRdsClient()
	if err != nil {
		return false, fmt.Errorf("unable to create RDS client, %v", err)
	}

	clusterID := remoteCentralNamespace + "-db-cluster"
	instanceID := remoteCentralNamespace + "-db-instance"

	//TODO: do not skip taking a final DB snapshot
	if instanceExists(rdsClient, instanceID) {
		//TODO: don't delete if state is "deleting"
		_, err = rdsClient.DeleteDBInstance(newDeleteCentralDBInstanceInput(instanceID, true))
		if err != nil {
			return false, fmt.Errorf("deleting DB instance: %v", err)
		}
	}

	if clusterExists(rdsClient, clusterID) {
		//TODO: don't delete if state is "deleting"
		_, err = rdsClient.DeleteDBCluster(newDeleteCentralDBClusterInput(clusterID, true))
		if err != nil {
			return false, fmt.Errorf("deleting DB cluster: %v", err)
		}
	}

	return true, nil
}

func newRdsClient() (*rds.RDS, error) {
	cfg := &aws.Config{
		Region: aws.String(awsRegion),
	}

	sess, err := session.NewSession(cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create session, %v", err)
	}

	return rds.New(sess), nil
}

func ensureDBClusterCreated(ctx context.Context, client ctrlClient.Client, rdsClient *rds.RDS, clusterID string, remoteCentralNamespace string) error {
	if !clusterExists(rdsClient, clusterID) {
		// cluster does not exist, create it
		glog.Infof("Provisioning RDS database cluster.")
		dbPassword, err := getDBPassword(ctx, client, remoteCentralNamespace)
		if err != nil {
			return fmt.Errorf("getting password for DB cluster: %v", err)
		}
		_, err = rdsClient.CreateDBCluster(newCreateCentralDBClusterInput(clusterID, dbPassword))
		if err != nil {
			return fmt.Errorf("creating DB cluster: %v", err)
		}
	}

	return nil
}

func ensureDBInstanceCreated(rdsClient *rds.RDS, instanceID string, clusterID string) error {
	if !instanceExists(rdsClient, instanceID) {
		// instance does not exist, create it
		glog.Infof("Provisioning RDS database instance.")
		_, err := rdsClient.CreateDBInstance(newCreateCentralDBInstanceInput(clusterID, instanceID))
		if err != nil {
			// TODO: delete cluster
			return fmt.Errorf("creating DB instance: %v", err)
		}
	}

	return nil
}

func clusterExists(rdsClient *rds.RDS, clusterID string) bool {
	dbClusterQuery := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(clusterID),
	}

	_, err := rdsClient.DescribeDBClusters(dbClusterQuery)
	return err == nil
}

func instanceExists(rdsClient *rds.RDS, instanceID string) bool {
	dbInstanceQuery := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(instanceID),
		//TODO: add cluster Filter
	}

	_, err := rdsClient.DescribeDBInstances(dbInstanceQuery)
	return err == nil
}

func waitForInstanceToBeAvailable(rdsClient *rds.RDS, instanceID string, clusterID string) (string, error) {
	dbInstanceQuery := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(instanceID),
		//TODO: add cluster Filter
	}

	dbClusterQuery := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(clusterID),
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
		if dbInstanceStatus == dbAvailableStatus {
			clusterResult, err := rdsClient.DescribeDBClusters(dbClusterQuery)
			if err != nil {
				return "", fmt.Errorf("retrieving DB cluster description: %v", err)
			}

			connectionString := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=require",
				*clusterResult.DBClusters[0].Endpoint, 5432, dbUser, "postgres")

			return connectionString, nil
		}

		glog.Infof("RDS instance status: %s", dbInstanceStatus)
		time.Sleep(10 * time.Second)
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

func newCreateCentralDBClusterInput(clusterID string, dbPassword string) *rds.CreateDBClusterInput {
	return &rds.CreateDBClusterInput{
		DBClusterIdentifier: aws.String(clusterID),
		Engine:              aws.String(dbEngine),
		EngineVersion:       aws.String(dbEngineVersion),
		MasterUsername:      aws.String(dbUser),
		MasterUserPassword:  aws.String(dbPassword),
		// TODO: the security group needs to be created during the data plane terraforming, and made known to
		// fleet-manager via a configuration parameter. I created this one in the AWS Console.
		VpcSecurityGroupIds: aws.StringSlice([]string{"sg-04dcc23a03646041c"}),
		ServerlessV2ScalingConfiguration: &rds.ServerlessV2ScalingConfiguration{
			MinCapacity: aws.Float64(0.5),
			MaxCapacity: aws.Float64(16),
		},
		BackupRetentionPeriod: aws.Int64(30),
		StorageEncrypted:      aws.Bool(true),

		// TODO: The following are some extra parameters to consider
		// AvailabilityZones: // TODO: determine the AZ in which the Central is running
		// EnableCloudwatchLogsExports // TODO: enable
		// Tags // TODO: e.g. could add a tag that allows us to identify the associated Central
		// PreferredBackupWindow
		// PreferredMaintenanceWindow
	}
}

func newCreateCentralDBInstanceInput(clusterID, instanceID string) *rds.CreateDBInstanceInput {
	return &rds.CreateDBInstanceInput{
		DBInstanceClass:      aws.String(dbInstanceClass),
		DBClusterIdentifier:  aws.String(clusterID),
		DBInstanceIdentifier: aws.String(instanceID),
		Engine:               aws.String(dbEngine),
		PubliclyAccessible:   aws.Bool(true), // TODO: should be false
	}
}

func newDeleteCentralDBInstanceInput(instanceID string, skipFinalSnapshot bool) *rds.DeleteDBInstanceInput {
	return &rds.DeleteDBInstanceInput{
		DBInstanceIdentifier: aws.String(instanceID),
		SkipFinalSnapshot:    aws.Bool(skipFinalSnapshot),
	}
}

func newDeleteCentralDBClusterInput(clusterID string, skipFinalSnapshot bool) *rds.DeleteDBClusterInput {
	return &rds.DeleteDBClusterInput{
		DBClusterIdentifier: aws.String(clusterID),
		SkipFinalSnapshot:   aws.Bool(skipFinalSnapshot),
	}
}
