package dbprovisioning

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awscredentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	dbAvailableStatus = "available"
	dbDeletingStatus  = "deleting"

	awsRegion        = "us-east-1" // TODO: this should not be hardcoded
	dbEngine         = "aurora-postgresql"
	dbEngineVersion  = "13.7"
	dbInstanceClass  = "db.serverless"
	dbUser           = "rhacs_master"
	dbInstanceSuffix = "-db-instance"
	dbClusterSuffix  = "-db-cluster"
)

// RDSProvisioningClient is an AWS RDS client tied to one Central instance. It provisions and deprovisions databases
// for the Central.
type RDSProvisioningClient struct {
	dbSecurityGroup string
	dbSubnetGroup   string

	client    ctrlClient.Client
	rdsClient *rds.RDS
}

// AWSCredentials stores the credentials for the AWS RDS API.
type AWSCredentials struct {
	// AccessKeyID is the AWS access key identifier.
	AccessKeyID string
	// SecretAccessKey is the AWS secret access key.
	SecretAccessKey string
	// SessionToken is a token required for temporary security credentials retrieved via STS.
	SessionToken string
}

// EnsureDBProvisioned is a blocking function that makes sure that an RDS database was provisioned for a Central
func (c *RDSProvisioningClient) EnsureDBProvisioned(ctx context.Context, centralNamespace, centralDbSecretName string) (string, error) {
	clusterID := centralNamespace + dbClusterSuffix
	instanceID := centralNamespace + dbInstanceSuffix

	err := c.ensureDBClusterCreated(ctx, clusterID, centralNamespace, centralDbSecretName)
	if err != nil {
		return "", fmt.Errorf("ensuring DB cluster %s exists: %v", clusterID, err)
	}

	err = c.ensureDBInstanceCreated(instanceID, clusterID)
	if err != nil {
		return "", fmt.Errorf("ensuring DB instance %s exists in cluster %s: %v", instanceID, clusterID, err)
	}

	return c.waitForInstanceToBeAvailable(instanceID, clusterID)
}

// EnsureDBDeprovisioned is a function that initiates the deprovisioning of the RDS database of a Central
// Unlike EnsureDBProvisioned, this function does not block until the DB is deprovisioned
func (c *RDSProvisioningClient) EnsureDBDeprovisioned(centralNamespace string) (bool, error) {
	clusterID := centralNamespace + dbClusterSuffix
	instanceID := centralNamespace + dbInstanceSuffix

	if c.instanceExists(instanceID) {
		status, err := c.instanceStatus(instanceID)
		if err != nil {
			return false, fmt.Errorf("getting DB instance status: %v", err)
		}
		if status != dbDeletingStatus {
			//TODO: do not skip taking a final DB snapshot
			glog.Infof("Deprovisioning RDS database instance.")
			_, err := c.rdsClient.DeleteDBInstance(newDeleteCentralDBInstanceInput(instanceID, true))
			if err != nil {
				return false, fmt.Errorf("deleting DB instance: %v", err)
			}
		}
	}

	if c.clusterExists(clusterID) {
		status, err := c.clusterStatus(clusterID)
		if err != nil {
			return false, fmt.Errorf("getting DB cluster status: %v", err)
		}
		if status != dbDeletingStatus {
			//TODO: do not skip taking a final DB snapshot
			glog.Infof("Deprovisioning RDS database cluster.")
			_, err := c.rdsClient.DeleteDBCluster(newDeleteCentralDBClusterInput(clusterID, true))
			if err != nil {
				return false, fmt.Errorf("deleting DB cluster: %v", err)
			}
		}
	}

	return true, nil
}

func (c *RDSProvisioningClient) ensureDBClusterCreated(ctx context.Context, clusterID, centralNamespace, centralDbSecretName string) error {
	if !c.clusterExists(clusterID) {
		// cluster does not exist, create it
		glog.Infof("Provisioning RDS database cluster.")
		dbPassword, err := c.getDBPassword(ctx, centralNamespace, centralDbSecretName)
		if err != nil {
			return fmt.Errorf("getting password for DB cluster: %v", err)
		}
		_, err = c.rdsClient.CreateDBCluster(newCreateCentralDBClusterInput(clusterID, dbPassword, c.dbSecurityGroup, c.dbSubnetGroup))
		if err != nil {
			return fmt.Errorf("creating DB cluster: %v", err)
		}
	}

	return nil
}

func (c *RDSProvisioningClient) ensureDBInstanceCreated(instanceID string, clusterID string) error {
	if !c.instanceExists(instanceID) {
		// instance does not exist, create it
		glog.Infof("Provisioning RDS database instance.")
		_, err := c.rdsClient.CreateDBInstance(newCreateCentralDBInstanceInput(clusterID, instanceID))
		if err != nil {
			// TODO: delete cluster if instance cannot be created?
			return fmt.Errorf("creating DB instance: %v", err)
		}
	}

	return nil
}

func (c *RDSProvisioningClient) clusterExists(clusterID string) bool {
	dbClusterQuery := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(clusterID),
	}

	_, err := c.rdsClient.DescribeDBClusters(dbClusterQuery)
	return err == nil
}

func (c *RDSProvisioningClient) instanceExists(instanceID string) bool {
	dbInstanceQuery := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(instanceID),
	}

	_, err := c.rdsClient.DescribeDBInstances(dbInstanceQuery)
	return err == nil
}

func (c *RDSProvisioningClient) clusterStatus(clusterID string) (string, error) {
	dbClusterQuery := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(clusterID),
	}

	clusterResult, err := c.rdsClient.DescribeDBClusters(dbClusterQuery)
	if err != nil {
		return "", fmt.Errorf("getting cluster status: %v", err)
	}

	return *clusterResult.DBClusters[0].Status, nil
}

func (c *RDSProvisioningClient) instanceStatus(instanceID string) (string, error) {
	dbInstanceQuery := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(instanceID),
	}

	instanceResult, err := c.rdsClient.DescribeDBInstances(dbInstanceQuery)
	if err != nil {
		return "", fmt.Errorf("getting instance status: %v", err)
	}

	return *instanceResult.DBInstances[0].DBInstanceStatus, nil
}

func (c *RDSProvisioningClient) waitForInstanceToBeAvailable(instanceID string, clusterID string) (string, error) {
	dbInstanceQuery := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(instanceID),
	}

	dbClusterQuery := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(clusterID),
	}

	// TODO: implement a timeout for this loop
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

func (c *RDSProvisioningClient) getDBPassword(ctx context.Context, centralNamespace, centralDbSecretName string) (string, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: centralDbSecretName,
		},
	}
	err := c.client.Get(ctx, ctrlClient.ObjectKey{Namespace: centralNamespace, Name: centralDbSecretName}, secret)
	if err != nil {
		return "", fmt.Errorf("getting Central DB password from secret: %v", err)
	}

	if dbPassword, ok := secret.Data["password"]; ok {
		return string(dbPassword), nil
	}

	return "", fmt.Errorf("central DB secret does not contain password field: %v", err)
}

// NewRDSProvisioningClient initializes a new dbprovisioning.RDSProvisioningClient
func NewRDSProvisioningClient(dbSecurityGroup, dbSubnetGroup string, credentials AWSCredentials, client ctrlClient.Client) (*RDSProvisioningClient, error) {
	rdsClient, err := newRdsClient(credentials.AccessKeyID, credentials.SecretAccessKey, credentials.SessionToken)
	if err != nil {
		return nil, fmt.Errorf("unable to create RDS client, %v", err)
	}

	return &RDSProvisioningClient{
		rdsClient:       rdsClient,
		dbSecurityGroup: dbSecurityGroup,
		dbSubnetGroup:   dbSubnetGroup,
		client:          client,
	}, nil
}

func newCreateCentralDBClusterInput(clusterID, dbPassword, securityGroup, subnetGroup string) *rds.CreateDBClusterInput {
	return &rds.CreateDBClusterInput{
		DBClusterIdentifier: aws.String(clusterID),
		Engine:              aws.String(dbEngine),
		EngineVersion:       aws.String(dbEngineVersion),
		MasterUsername:      aws.String(dbUser),
		MasterUserPassword:  aws.String(dbPassword),
		VpcSecurityGroupIds: aws.StringSlice([]string{securityGroup}),
		DBSubnetGroupName:   aws.String(subnetGroup),
		ServerlessV2ScalingConfiguration: &rds.ServerlessV2ScalingConfiguration{
			MinCapacity: aws.Float64(0.5),
			MaxCapacity: aws.Float64(16),
		},
		BackupRetentionPeriod: aws.Int64(30),
		StorageEncrypted:      aws.Bool(true),
		// AvailabilityZones: // TODO: determine the AZ in which the Central is running
		// EnableCloudwatchLogsExports // TODO: enable
		// Tags // TODO: could add a tag that allows us to identify the associated Central
	}
}

func newCreateCentralDBInstanceInput(clusterID, instanceID string) *rds.CreateDBInstanceInput {
	return &rds.CreateDBInstanceInput{
		DBInstanceClass:      aws.String(dbInstanceClass),
		DBClusterIdentifier:  aws.String(clusterID),
		DBInstanceIdentifier: aws.String(instanceID),
		Engine:               aws.String(dbEngine),
		PubliclyAccessible:   aws.Bool(true), // TODO: must be set to false after VPC peering is done
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

func newRdsClient(accessKeyID, secretAccessKey, sessionToken string) (*rds.RDS, error) {
	cfg := &aws.Config{
		Region: aws.String(awsRegion),
		Credentials: awscredentials.NewStaticCredentials(
			accessKeyID,
			secretAccessKey,
			sessionToken),
	}

	sess, err := session.NewSession(cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create session, %v", err)
	}

	return rds.New(sess), nil
}
