// Package rdsclient provides functionality to provision and deprovision RDS DB instances on AWS
package rdsclient

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	dbAvailableStatus = "available"

	awsRegion        = "us-east-1" // TODO: this should not be hardcoded
	dbEngine         = "aurora-postgresql"
	dbEngineVersion  = "13.7"
	dbInstanceClass  = "db.serverless"
	dbUser           = "rhacs_master"
	dbInstanceSuffix = "-db-instance"
	dbClusterSuffix  = "-db-cluster"
)

// Client is an AWS RDS client tied to one Central instance. It provisions and deprovisions databases
// for the Central.
type Client struct {
	centralDbSecretName string
	centralNamespace    string

	rdsClient *rds.RDS
}

// EnsureDBProvisioned is a blocking function that makes sure that an RDS database was provisioned for a Central
func (c *Client) EnsureDBProvisioned(ctx context.Context, client ctrlClient.Client) (string, error) {
	clusterID := c.centralNamespace + dbClusterSuffix
	instanceID := c.centralNamespace + dbInstanceSuffix

	err := c.ensureDBClusterCreated(ctx, client, clusterID)
	if err != nil {
		return "", fmt.Errorf("ensuring DB cluster %s exists", clusterID)
	}

	err = c.ensureDBInstanceCreated(instanceID, clusterID)
	if err != nil {
		return "", fmt.Errorf("ensuring DB instance %s exists in cluster %s", instanceID, clusterID)
	}

	return c.waitForInstanceToBeAvailable(instanceID, clusterID)
}

// EnsureDBDeprovisioned is a function that initiates the deprovisioning of the RDS database of a Central
// Unlike EnsureDBProvisioned, this function does not block until the DB is deprovisioned
func (c *Client) EnsureDBDeprovisioned() (bool, error) {
	clusterID := c.centralNamespace + dbClusterSuffix
	instanceID := c.centralNamespace + dbInstanceSuffix

	//TODO: do not skip taking a final DB snapshot
	if c.instanceExists(instanceID) {
		//TODO: don't delete if state is "deleting"
		_, err := c.rdsClient.DeleteDBInstance(newDeleteCentralDBInstanceInput(instanceID, true))
		if err != nil {
			return false, fmt.Errorf("deleting DB instance: %v", err)
		}
	}

	if c.clusterExists(clusterID) {
		//TODO: don't delete if state is "deleting"
		_, err := c.rdsClient.DeleteDBCluster(newDeleteCentralDBClusterInput(clusterID, true))
		if err != nil {
			return false, fmt.Errorf("deleting DB cluster: %v", err)
		}
	}

	return true, nil
}

func (c *Client) ensureDBClusterCreated(ctx context.Context, client ctrlClient.Client, clusterID string) error {
	if !c.clusterExists(clusterID) {
		// cluster does not exist, create it
		glog.Infof("Provisioning RDS database cluster.")
		dbPassword, err := c.getDBPassword(ctx, client, c.centralNamespace)
		if err != nil {
			return fmt.Errorf("getting password for DB cluster: %v", err)
		}
		_, err = c.rdsClient.CreateDBCluster(newCreateCentralDBClusterInput(clusterID, dbPassword))
		if err != nil {
			return fmt.Errorf("creating DB cluster: %v", err)
		}
	}

	return nil
}

func (c *Client) ensureDBInstanceCreated(instanceID string, clusterID string) error {
	if !c.instanceExists(instanceID) {
		// instance does not exist, create it
		glog.Infof("Provisioning RDS database instance.")
		_, err := c.rdsClient.CreateDBInstance(newCreateCentralDBInstanceInput(clusterID, instanceID))
		if err != nil {
			// TODO: delete cluster
			return fmt.Errorf("creating DB instance: %v", err)
		}
	}

	return nil
}

func (c *Client) clusterExists(clusterID string) bool {
	dbClusterQuery := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(clusterID),
	}

	_, err := c.rdsClient.DescribeDBClusters(dbClusterQuery)
	return err == nil
}

func (c *Client) instanceExists(instanceID string) bool {
	dbInstanceQuery := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(instanceID),
		//TODO: add cluster Filter
	}

	_, err := c.rdsClient.DescribeDBInstances(dbInstanceQuery)
	return err == nil
}

func (c *Client) waitForInstanceToBeAvailable(instanceID string, clusterID string) (string, error) {
	dbInstanceQuery := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(instanceID),
		//TODO: add cluster Filter
	}

	dbClusterQuery := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(clusterID),
	}

	for {
		result, err := c.rdsClient.DescribeDBInstances(dbInstanceQuery)
		if err != nil {
			return "", fmt.Errorf("retrieving DB instance state: %v", err)
		}

		if len(result.DBInstances) != 1 {
			return "", fmt.Errorf("unexpected number of DB instances: %v", err)
		}

		dbInstanceStatus := *result.DBInstances[0].DBInstanceStatus
		if dbInstanceStatus == dbAvailableStatus {
			clusterResult, err := c.rdsClient.DescribeDBClusters(dbClusterQuery)
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

func (c *Client) getDBPassword(ctx context.Context, client ctrlClient.Client, remoteCentralNamespace string) (string, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: c.centralDbSecretName,
		},
	}
	err := client.Get(ctx, ctrlClient.ObjectKey{Namespace: remoteCentralNamespace, Name: c.centralDbSecretName}, secret)
	if err != nil {
		return "", fmt.Errorf("getting Central DB password from secret: %v", err)
	}

	if dbPassword, ok := secret.Data["password"]; ok {
		return string(dbPassword), nil
	}

	return "", fmt.Errorf("central DB secret does not contain password field: %v", err)
}

// NewClient initializes a new rdsclient.Client
func NewClient(centralDbSecretName string, centralNamespace string) (*Client, error) {
	rdsClient, err := newRdsClient()
	if err != nil {
		return nil, fmt.Errorf("unable to create RDS client, %v", err)
	}

	return &Client{
		centralDbSecretName: centralDbSecretName, // pragma: allowlist secret
		centralNamespace:    centralNamespace,
		rdsClient:           rdsClient,
	}, nil
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
