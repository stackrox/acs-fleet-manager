// Package awsclient provides AWS-specific implementations of the interfaces in cloudprovider
package awsclient

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	awscredentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/golang/glog"
)

const (
	dbAvailableStatus = "available"
	dbDeletingStatus  = "deleting"

	dbEngine         = "aurora-postgresql"
	dbEngineVersion  = "13.7"
	dbInstanceClass  = "db.serverless"
	dbUser           = "rhacs_master"
	dbInstanceSuffix = "-db-instance"
	dbClusterSuffix  = "-db-cluster"
	dbPostgresPort   = 5432
	dbName           = "postgres"
)

// RDS is an AWS RDS client tied to one Central instance. It provisions and deprovisions databases
// for the Central.
type RDS struct {
	dbSecurityGroup string
	dbSubnetGroup   string

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
func (r *RDS) EnsureDBProvisioned(ctx context.Context, databaseID, masterPassword string) (string, error) {
	clusterID := getClusterID(databaseID)
	instanceID := getInstanceID(databaseID)

	if err := r.ensureDBClusterCreated(clusterID, masterPassword); err != nil {
		return "", fmt.Errorf("ensuring DB cluster %s exists: %w", clusterID, err)
	}

	if err := r.ensureDBInstanceCreated(instanceID, clusterID); err != nil {
		return "", fmt.Errorf("ensuring DB instance %s exists in cluster %s: %w", instanceID, clusterID, err)
	}

	return r.waitForInstanceToBeAvailable(ctx, instanceID, clusterID)
}

// EnsureDBDeprovisioned is a function that initiates the deprovisioning of the RDS database of a Central
// Unlike EnsureDBProvisioned, this function does not block until the DB is deprovisioned
func (r *RDS) EnsureDBDeprovisioned(databaseID string) (bool, error) {
	clusterID := getClusterID(databaseID)
	instanceID := getInstanceID(databaseID)

	instanceExists, err := r.instanceExists(instanceID)
	if err != nil {
		return false, fmt.Errorf("checking if DB instance exists: %w", err)
	}
	if instanceExists {
		status, err := r.instanceStatus(instanceID)
		if err != nil {
			return false, fmt.Errorf("getting DB instance status: %w", err)
		}
		if status != dbDeletingStatus {
			glog.Infof("Initiating deprovisioning of RDS database instance %s.", instanceID)
			// TODO(ROX-13692): do not skip taking a final DB snapshot
			_, err := r.rdsClient.DeleteDBInstance(newDeleteCentralDBInstanceInput(instanceID, true))
			if err != nil {
				return false, fmt.Errorf("deleting DB instance: %w", err)
			}
		}
	}

	clusterExists, err := r.clusterExists(clusterID)
	if err != nil {
		return false, fmt.Errorf("checking if DB cluster exists: %w", err)
	}
	if clusterExists {
		status, err := r.clusterStatus(clusterID)
		if err != nil {
			return false, fmt.Errorf("getting DB cluster status: %w", err)
		}
		if status != dbDeletingStatus {
			glog.Infof("Initiating deprovisioning of RDS database cluster %s.", clusterID)
			// TODO(ROX-13692): do not skip taking a final DB snapshot
			_, err := r.rdsClient.DeleteDBCluster(newDeleteCentralDBClusterInput(clusterID, true))
			if err != nil {
				return false, fmt.Errorf("deleting DB cluster: %w", err)
			}
		}
	}

	return true, nil
}

func (r *RDS) ensureDBClusterCreated(clusterID, masterPassword string) error {
	clusterExists, err := r.clusterExists(clusterID)
	if err != nil {
		return fmt.Errorf("checking if DB cluster exists: %w", err)
	}
	if clusterExists {
		return nil
	}

	glog.Infof("Initiating provisioning of RDS database cluster %s.", clusterID)
	_, err = r.rdsClient.CreateDBCluster(newCreateCentralDBClusterInput(clusterID, masterPassword, r.dbSecurityGroup, r.dbSubnetGroup))
	if err != nil {
		return fmt.Errorf("creating DB cluster: %w", err)
	}

	return nil
}

func (r *RDS) ensureDBInstanceCreated(instanceID string, clusterID string) error {
	instanceExists, err := r.instanceExists(instanceID)
	if err != nil {
		return fmt.Errorf("checking if DB instance exists: %w", err)
	}
	if instanceExists {
		return nil
	}

	glog.Infof("Initiating provisioning of RDS database instance %s.", instanceID)
	_, err = r.rdsClient.CreateDBInstance(newCreateCentralDBInstanceInput(clusterID, instanceID))
	if err != nil {
		return fmt.Errorf("creating DB instance: %w", err)
	}

	return nil
}

func (r *RDS) clusterExists(clusterID string) (bool, error) {
	if _, err := r.describeDBCluster(clusterID); err != nil {
		var aerr awserr.Error
		if errors.As(err, &aerr) {
			switch aerr.Code() {
			case rds.ErrCodeDBClusterNotFoundFault:
				return false, nil
			}
		}
		return false, err
	}

	return true, nil
}

func (r *RDS) instanceExists(instanceID string) (bool, error) {
	if _, err := r.describeDBInstance(instanceID); err != nil {
		var aerr awserr.Error
		if errors.As(err, &aerr) {
			switch aerr.Code() {
			case rds.ErrCodeDBInstanceNotFoundFault:
				return false, nil
			}
		}
		return false, err
	}

	return true, nil
}

func (r *RDS) clusterStatus(clusterID string) (string, error) {
	dbCluster, err := r.describeDBCluster(clusterID)
	if err != nil {
		return "", err
	}

	return *dbCluster.Status, nil
}

func (r *RDS) instanceStatus(instanceID string) (string, error) {
	dbInstance, err := r.describeDBInstance(instanceID)
	if err != nil {
		return "", err
	}

	return *dbInstance.DBInstanceStatus, nil
}

func (r *RDS) describeDBInstance(instanceID string) (*rds.DBInstance, error) {
	result, err := r.rdsClient.DescribeDBInstances(
		&rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: aws.String(instanceID),
		})
	if err != nil {
		return nil, fmt.Errorf("retrieving DB instance state: %w", err)
	}

	if len(result.DBInstances) != 1 {
		// this should never happen (DescribeDBInstances should return either 1 instance, or ErrCodeDBInstanceNotFoundFault)
		return nil, fmt.Errorf("unexpected number of DB instances: %d", len(result.DBInstances))
	}

	return result.DBInstances[0], nil
}

func (r *RDS) describeDBCluster(clusterID string) (*rds.DBCluster, error) {
	result, err := r.rdsClient.DescribeDBClusters(&rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(clusterID),
	})
	if err != nil {
		return nil, fmt.Errorf("retrieving DB cluster description: %w", err)
	}

	if len(result.DBClusters) != 1 {
		// this should never happen (DescribeDBClusters should return either 1 cluster, or ErrCodeDBClusterNotFoundFault)
		return nil, fmt.Errorf("unexpected number of DB clusters: %d", len(result.DBClusters))
	}

	return result.DBClusters[0], nil
}

func (r *RDS) waitForInstanceToBeAvailable(ctx context.Context, instanceID string, clusterID string) (string, error) {
	for {
		dbInstanceStatus, err := r.instanceStatus(instanceID)
		if err != nil {
			return "", err
		}

		if dbInstanceStatus == dbAvailableStatus {
			dbCluster, err := r.describeDBCluster(clusterID)
			if err != nil {
				return "", err
			}

			connectionString := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=require",
				*dbCluster.Endpoint, dbPostgresPort, dbUser, dbName)

			return connectionString, nil
		}

		glog.Infof("RDS instance status: %s", dbInstanceStatus)
		ticker := time.NewTicker(10 * time.Second)
		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return "", fmt.Errorf("waiting for RDS instance to be available: %w", ctx.Err())
		}
	}
}

// NewRDSClient initializes a new awsclient.RDS
func NewRDSClient(awsRegion, dbSecurityGroup, dbSubnetGroup string, credentials AWSCredentials) (*RDS, error) {
	rdsClient, err := newRdsClient(awsRegion, credentials.AccessKeyID, credentials.SecretAccessKey, credentials.SessionToken)
	if err != nil {
		return nil, fmt.Errorf("unable to create RDS client, %w", err)
	}

	return &RDS{
		rdsClient:       rdsClient,
		dbSecurityGroup: dbSecurityGroup,
		dbSubnetGroup:   dbSubnetGroup,
	}, nil
}

func getClusterID(databaseID string) string {
	return databaseID + dbClusterSuffix
}

func getInstanceID(databaseID string) string {
	return databaseID + dbInstanceSuffix
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

func newRdsClient(awsRegion, accessKeyID, secretAccessKey, sessionToken string) (*rds.RDS, error) {
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
