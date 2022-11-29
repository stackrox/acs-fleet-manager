// Package awsclient provides AWS-specific implementations of the interfaces in cloudprovider
package awsclient

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

// RDS is an AWS RDS client tied to one Central instance. It provisions and deprovisions databases
// for the Central.
type RDS struct {
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
func (r *RDS) EnsureDBProvisioned(ctx context.Context, centralNamespace, centralDbSecretName string) (string, error) {
	clusterID := centralNamespace + dbClusterSuffix
	instanceID := centralNamespace + dbInstanceSuffix

	if err := r.ensureDBClusterCreated(ctx, clusterID, centralNamespace, centralDbSecretName); err != nil {
		return "", fmt.Errorf("ensuring DB cluster %s exists: %w", clusterID, err)
	}

	if err := r.ensureDBInstanceCreated(instanceID, clusterID); err != nil {
		return "", fmt.Errorf("ensuring DB instance %s exists in cluster %s: %w", instanceID, clusterID, err)
	}

	return r.waitForInstanceToBeAvailable(ctx, instanceID, clusterID)
}

// EnsureDBDeprovisioned is a function that initiates the deprovisioning of the RDS database of a Central
// Unlike EnsureDBProvisioned, this function does not block until the DB is deprovisioned
func (r *RDS) EnsureDBDeprovisioned(centralNamespace string) (bool, error) {
	clusterID := centralNamespace + dbClusterSuffix
	instanceID := centralNamespace + dbInstanceSuffix

	if r.instanceExists(instanceID) {
		status, err := r.instanceStatus(instanceID)
		if err != nil {
			return false, fmt.Errorf("getting DB instance status: %w", err)
		}
		if status != dbDeletingStatus {
			//TODO: do not skip taking a final DB snapshot
			glog.Infof("Deprovisioning RDS database instance.")
			_, err := r.rdsClient.DeleteDBInstance(newDeleteCentralDBInstanceInput(instanceID, true))
			if err != nil {
				return false, fmt.Errorf("deleting DB instance: %w", err)
			}
		}
	}

	if r.clusterExists(clusterID) {
		status, err := r.clusterStatus(clusterID)
		if err != nil {
			return false, fmt.Errorf("getting DB cluster status: %w", err)
		}
		if status != dbDeletingStatus {
			//TODO: do not skip taking a final DB snapshot
			glog.Infof("Deprovisioning RDS database cluster.")
			_, err := r.rdsClient.DeleteDBCluster(newDeleteCentralDBClusterInput(clusterID, true))
			if err != nil {
				return false, fmt.Errorf("deleting DB cluster: %w", err)
			}
		}
	}

	return true, nil
}

func (r *RDS) ensureDBClusterCreated(ctx context.Context, clusterID, centralNamespace, centralDbSecretName string) error {
	if r.clusterExists(clusterID) {
		return nil
	}

	dbPassword, err := r.getDBPassword(ctx, centralNamespace, centralDbSecretName)
	if err != nil {
		return fmt.Errorf("getting password for DB cluster: %w", err)
	}

	glog.Infof("Initiating provisioning of RDS database cluster %s.", clusterID)
	_, err = r.rdsClient.CreateDBCluster(newCreateCentralDBClusterInput(clusterID, dbPassword, r.dbSecurityGroup, r.dbSubnetGroup))
	if err != nil {
		return fmt.Errorf("creating DB cluster: %w", err)
	}

	return nil
}

func (r *RDS) ensureDBInstanceCreated(instanceID string, clusterID string) error {
	if r.instanceExists(instanceID) {
		return nil
	}

	glog.Infof("Initiating provisioning of RDS database instance %s.", instanceID)
	_, err := r.rdsClient.CreateDBInstance(newCreateCentralDBInstanceInput(clusterID, instanceID))
	if err != nil {
		return fmt.Errorf("creating DB instance: %w", err)
	}

	return nil
}

func (r *RDS) clusterExists(clusterID string) bool {
	dbClusterQuery := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(clusterID),
	}

	_, err := r.rdsClient.DescribeDBClusters(dbClusterQuery)
	return err == nil
}

func (r *RDS) instanceExists(instanceID string) bool {
	dbInstanceQuery := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(instanceID),
	}

	_, err := r.rdsClient.DescribeDBInstances(dbInstanceQuery)
	return err == nil
}

func (r *RDS) clusterStatus(clusterID string) (string, error) {
	dbClusterQuery := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(clusterID),
	}

	clusterResult, err := r.rdsClient.DescribeDBClusters(dbClusterQuery)
	if err != nil {
		return "", fmt.Errorf("getting cluster status: %w", err)
	}

	return *clusterResult.DBClusters[0].Status, nil
}

func (r *RDS) instanceStatus(instanceID string) (string, error) {
	dbInstanceQuery := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(instanceID),
	}

	instanceResult, err := r.rdsClient.DescribeDBInstances(dbInstanceQuery)
	if err != nil {
		return "", fmt.Errorf("getting instance status: %w", err)
	}

	return *instanceResult.DBInstances[0].DBInstanceStatus, nil
}

func (r *RDS) waitForInstanceToBeAvailable(ctx context.Context, instanceID string, clusterID string) (string, error) {
	dbInstanceQuery := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(instanceID),
	}

	dbClusterQuery := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(clusterID),
	}

	for {
		if ctx.Err() != nil {
			return "", fmt.Errorf("waiting for RDS instance to be available: %w", ctx.Err())
		}

		dbInstanceStatuses, err := r.rdsClient.DescribeDBInstances(dbInstanceQuery)
		if err != nil {
			return "", fmt.Errorf("retrieving DB instance state: %w", err)
		}

		dbInstanceStatus := *dbInstanceStatuses.DBInstances[0].DBInstanceStatus
		if dbInstanceStatus == dbAvailableStatus {
			dbClusterStatus, err := r.rdsClient.DescribeDBClusters(dbClusterQuery)
			if err != nil {
				return "", fmt.Errorf("retrieving DB cluster description: %w", err)
			}

			connectionString := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=require",
				*dbClusterStatus.DBClusters[0].Endpoint, 5432, dbUser, "postgres")

			return connectionString, nil
		}

		glog.Infof("RDS instance status: %s", dbInstanceStatus)
		time.Sleep(10 * time.Second)
	}
}

func (r *RDS) getDBPassword(ctx context.Context, centralNamespace, centralDbSecretName string) (string, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: centralDbSecretName,
		},
	}
	err := r.client.Get(ctx, ctrlClient.ObjectKey{Namespace: centralNamespace, Name: centralDbSecretName}, secret)
	if err != nil {
		return "", fmt.Errorf("getting Central DB password from secret: %w", err)
	}

	if dbPassword, ok := secret.Data["password"]; ok {
		return string(dbPassword), nil
	}

	return "", fmt.Errorf("central DB secret does not contain password field: %w", err)
}

// NewRDSClient initializes a new awsclient.RDS
func NewRDSClient(dbSecurityGroup, dbSubnetGroup string, credentials AWSCredentials, client ctrlClient.Client) (*RDS, error) {
	rdsClient, err := newRdsClient(credentials.AccessKeyID, credentials.SecretAccessKey, credentials.SessionToken)
	if err != nil {
		return nil, fmt.Errorf("unable to create RDS client, %w", err)
	}

	return &RDS{
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
		PubliclyAccessible:   aws.Bool(false),
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
		return nil, fmt.Errorf("unable to create session, %w", err)
	}

	return rds.New(sess), nil
}
