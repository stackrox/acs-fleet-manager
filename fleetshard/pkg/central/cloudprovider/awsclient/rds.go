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
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/postgres"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
)

const (
	dbAvailableStatus = "available"
	dbDeletingStatus  = "deleting"

	dbUser           = "rhacs_master"
	dbPrefix         = "rhacs-"
	dbInstanceSuffix = "-db-instance"
	dbFailoverSuffix = "-db-failover"
	dbClusterSuffix  = "-db-cluster"
	awsRetrySeconds  = 30

	// DB cluster / instance configuration parameters
	dbEngine                = "aurora-postgresql"
	dbEngineVersion         = "13.9" // 13.9 is a LTS Aurora PostgreSQL version
	dbAutoVersionUpgrade    = false  // disable auto upgrades while on LTS version (see ROX-16099)
	dbInstanceClass         = "db.serverless"
	dbPostgresPort          = 5432
	dbName                  = "postgres"
	dbBackupRetentionPeriod = 30
	dbInstancePromotionTier = 2 // a tier of 2 (or higher) ensures that readers and writers can scale independently
	dbCACertificateType     = "rds-ca-rsa4096-g1"
	dataplaneClusterNameKey = "DataplaneClusterName"

	// The Aurora Serverless v2 DB instance configuration in ACUs (Aurora Capacity Units)
	// 1 ACU = 1 vCPU + 2GB RAM
	dbMinCapacityACU = 0.5
	dbMaxCapacityACU = 16
)

// RDS is an AWS RDS client tied to one Central instance. It provisions and deprovisions databases
// for the Central.
type RDS struct {
	dbSecurityGroup      string
	dbSubnetGroup        string
	performanceInsights  bool
	dataplaneClusterName string

	rdsClient *rds.RDS
}

// EnsureDBProvisioned is a blocking function that makes sure that an RDS database was provisioned for a Central
func (r *RDS) EnsureDBProvisioned(ctx context.Context, databaseID, masterPassword string) error {
	clusterID := getClusterID(databaseID)
	if err := r.ensureDBClusterCreated(clusterID, masterPassword); err != nil {
		return fmt.Errorf("ensuring DB cluster %s exists: %w", clusterID, err)
	}

	instanceID := getInstanceID(databaseID)
	if err := r.ensureDBInstanceCreated(instanceID, clusterID); err != nil {
		return fmt.Errorf("ensuring DB instance %s exists in cluster %s: %w", instanceID, clusterID, err)
	}

	failoverID := getFailoverInstanceID(databaseID)
	if err := r.ensureDBInstanceCreated(failoverID, clusterID); err != nil {
		return fmt.Errorf("ensuring failover DB instance %s exists in cluster %s: %w", failoverID, clusterID, err)
	}

	return r.waitForInstanceToBeAvailable(ctx, instanceID)
}

// EnsureDBDeprovisioned is a function that initiates the deprovisioning of the RDS database of a Central
// Unlike EnsureDBProvisioned, this function does not block until the DB is deprovisioned
func (r *RDS) EnsureDBDeprovisioned(databaseID string) error {
	err := r.ensureInstanceDeleted(getInstanceID(databaseID))
	if err != nil {
		return err
	}

	err = r.ensureInstanceDeleted(getFailoverInstanceID(databaseID))
	if err != nil {
		return err
	}

	err = r.ensureClusterDeleted(getClusterID(databaseID))
	if err != nil {
		return err
	}

	return nil
}

// GetDBConnection returns a postgres.DBConnection struct, which contains the data necessary
// to construct a PostgreSQL connection string. It expects that the database was already provisioned.
func (r *RDS) GetDBConnection(databaseID string) (postgres.DBConnection, error) {
	dbCluster, err := r.describeDBCluster(getClusterID(databaseID))
	if err != nil {
		return postgres.DBConnection{}, err
	}

	connection, err := postgres.NewDBConnection(*dbCluster.Endpoint, dbPostgresPort, dbUser, dbName)
	if err != nil {
		return postgres.DBConnection{}, fmt.Errorf("incorrect DB connection parameters: %w", err)
	}

	return connection, nil
}

func (r *RDS) ensureDBClusterCreated(clusterID, masterPassword string) error {
	clusterExists, _, err := r.clusterStatus(clusterID)
	if err != nil {
		return fmt.Errorf("checking if DB cluster exists: %w", err)
	}
	if clusterExists {
		return nil
	}

	glog.Infof("Initiating provisioning of RDS database cluster %s.", clusterID)
	_, err = r.rdsClient.CreateDBCluster(newCreateCentralDBClusterInput(clusterID, masterPassword, r.dbSecurityGroup,
		r.dbSubnetGroup, r.dataplaneClusterName))
	if err != nil {
		return fmt.Errorf("creating DB cluster: %w", err)
	}

	return nil
}

func (r *RDS) ensureDBInstanceCreated(instanceID string, clusterID string) error {
	instanceExists, _, err := r.instanceStatus(instanceID)
	if err != nil {
		return fmt.Errorf("checking if DB instance exists: %w", err)
	}
	if instanceExists {
		return nil
	}

	glog.Infof("Initiating provisioning of RDS database instance %s.", instanceID)
	_, err = r.rdsClient.CreateDBInstance(newCreateCentralDBInstanceInput(clusterID, instanceID,
		r.dataplaneClusterName, r.performanceInsights))
	if err != nil {
		return fmt.Errorf("creating DB instance: %w", err)
	}

	return nil
}

func (r *RDS) ensureInstanceDeleted(instanceID string) error {
	instanceExists, instanceStatus, err := r.instanceStatus(instanceID)
	if err != nil {
		return fmt.Errorf("getting DB instance status: %w", err)
	}
	if !instanceExists {
		return nil
	}

	if instanceStatus != dbDeletingStatus {
		glog.Infof("Initiating deprovisioning of RDS database instance %s.", instanceID)
		_, err := r.rdsClient.DeleteDBInstance(newDeleteCentralDBInstanceInput(instanceID, true))
		if err != nil {
			return fmt.Errorf("deleting DB instance: %w", err)
		}
	}

	return nil
}

func (r *RDS) ensureClusterDeleted(clusterID string) error {
	clusterExists, clusterStatus, err := r.clusterStatus(clusterID)
	if err != nil {
		return fmt.Errorf("getting DB cluster status: %w", err)
	}
	if !clusterExists {
		return nil
	}

	if clusterStatus != dbDeletingStatus {
		glog.Infof("Initiating deprovisioning of RDS database cluster %s.", clusterID)
		_, err := r.rdsClient.DeleteDBCluster(newDeleteCentralDBClusterInput(clusterID, false))
		if err != nil {
			return fmt.Errorf("deleting DB cluster: %w", err)
		}
	}

	return nil
}

func (r *RDS) clusterStatus(clusterID string) (bool, string, error) {
	dbCluster, err := r.describeDBCluster(clusterID)
	if err != nil {
		var aerr awserr.Error
		if errors.As(err, &aerr) {
			switch aerr.Code() {
			case rds.ErrCodeDBClusterNotFoundFault:
				return false, "", nil
			}
		}
		return false, "", err
	}

	return true, *dbCluster.Status, nil
}

func (r *RDS) instanceStatus(instanceID string) (bool, string, error) {
	dbInstance, err := r.describeDBInstance(instanceID)
	if err != nil {
		var aerr awserr.Error
		if errors.As(err, &aerr) {
			switch aerr.Code() {
			case rds.ErrCodeDBInstanceNotFoundFault:
				return false, "", nil
			}
		}
		return false, "", err
	}

	return true, *dbInstance.DBInstanceStatus, nil
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

func (r *RDS) waitForInstanceToBeAvailable(ctx context.Context, instanceID string) error {
	for {
		dbInstanceExists, dbInstanceStatus, err := r.instanceStatus(instanceID)
		if err != nil {
			return err
		}

		if !dbInstanceExists {
			return fmt.Errorf("DB instance does not exist: %s", instanceID)
		}

		if dbInstanceStatus == dbAvailableStatus {
			return nil
		}

		glog.Infof("RDS instance status: %s (instance ID: %s)", dbInstanceStatus, instanceID)
		ticker := time.NewTicker(awsRetrySeconds * time.Second)
		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return fmt.Errorf("waiting for RDS instance to be available: %w", ctx.Err())
		}
	}
}

// NewRDSClient initializes a new awsclient.RDS
func NewRDSClient(config *config.Config, auth fleetmanager.Auth) (*RDS, error) {
	rdsClient, err := newRdsClient(config.AWS, auth)
	if err != nil {
		return nil, fmt.Errorf("unable to create RDS client: %w", err)
	}

	return &RDS{
		rdsClient:            rdsClient,
		dbSecurityGroup:      config.ManagedDB.SecurityGroup,
		dbSubnetGroup:        config.ManagedDB.SubnetGroup,
		performanceInsights:  config.ManagedDB.PerformanceInsights,
		dataplaneClusterName: config.ClusterName,
	}, nil
}

func getClusterID(databaseID string) string {
	return dbPrefix + databaseID + dbClusterSuffix
}

func getInstanceID(databaseID string) string {
	return dbPrefix + databaseID + dbInstanceSuffix
}

func getFailoverInstanceID(databaseID string) string {
	return dbPrefix + databaseID + dbFailoverSuffix
}

func newCreateCentralDBClusterInput(clusterID, dbPassword, securityGroup, subnetGroup, dataplaneClusterName string) *rds.CreateDBClusterInput {
	return &rds.CreateDBClusterInput{
		DBClusterIdentifier: aws.String(clusterID),
		Engine:              aws.String(dbEngine),
		EngineVersion:       aws.String(dbEngineVersion),
		MasterUsername:      aws.String(dbUser),
		MasterUserPassword:  aws.String(dbPassword),
		VpcSecurityGroupIds: aws.StringSlice([]string{securityGroup}),
		DBSubnetGroupName:   aws.String(subnetGroup),
		ServerlessV2ScalingConfiguration: &rds.ServerlessV2ScalingConfiguration{
			MinCapacity: aws.Float64(dbMinCapacityACU),
			MaxCapacity: aws.Float64(dbMaxCapacityACU),
		},
		BackupRetentionPeriod: aws.Int64(dbBackupRetentionPeriod),
		StorageEncrypted:      aws.Bool(true),
		Tags: []*rds.Tag{
			{
				Key:   aws.String(dataplaneClusterNameKey),
				Value: aws.String(dataplaneClusterName)},
		},
	}
}

func newCreateCentralDBInstanceInput(clusterID, instanceID, dataplaneClusterName string, performanceInsights bool) *rds.CreateDBInstanceInput {
	return &rds.CreateDBInstanceInput{
		DBInstanceClass:           aws.String(dbInstanceClass),
		DBClusterIdentifier:       aws.String(clusterID),
		DBInstanceIdentifier:      aws.String(instanceID),
		Engine:                    aws.String(dbEngine),
		PubliclyAccessible:        aws.Bool(false),
		EnablePerformanceInsights: aws.Bool(performanceInsights),
		PromotionTier:             aws.Int64(dbInstancePromotionTier),
		CACertificateIdentifier:   aws.String(dbCACertificateType),
		AutoMinorVersionUpgrade:   aws.Bool(dbAutoVersionUpgrade),
		Tags: []*rds.Tag{
			{
				Key:   aws.String(dataplaneClusterNameKey),
				Value: aws.String(dataplaneClusterName)},
		},
	}
}

func newDeleteCentralDBInstanceInput(instanceID string, skipFinalSnapshot bool) *rds.DeleteDBInstanceInput {
	return &rds.DeleteDBInstanceInput{
		DBInstanceIdentifier: aws.String(instanceID),
		SkipFinalSnapshot:    aws.Bool(skipFinalSnapshot),
	}
}

func newDeleteCentralDBClusterInput(clusterID string, skipFinalSnapshot bool) *rds.DeleteDBClusterInput {
	input := &rds.DeleteDBClusterInput{
		DBClusterIdentifier: aws.String(clusterID),
		SkipFinalSnapshot:   aws.Bool(skipFinalSnapshot),
	}

	if !skipFinalSnapshot {
		input.FinalDBSnapshotIdentifier = getFinalSnapshotID(clusterID)
	}

	return input
}

func newRdsClient(awsConfig config.AWS, auth fleetmanager.Auth) (*rds.RDS, error) {
	cfg := &aws.Config{
		Region: aws.String(awsConfig.Region),
	}
	sess, err := session.NewSession(cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create session for STS client: %w", err)
	}
	stsClient := sts.New(sess)

	roleProvider := stscreds.NewWebIdentityRoleProviderWithOptions(stsClient, awsConfig.RoleARN, "",
		&tokenFetcher{auth: auth})

	cfg.Credentials = awscredentials.NewCredentials(roleProvider)

	sess, err = session.NewSession(cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create session for RDS client: %w", err)
	}

	return rds.New(sess), nil
}

func getFinalSnapshotID(clusterID string) *string {
	return aws.String(fmt.Sprintf("%s-%s", clusterID, "final"))
}

type tokenFetcher struct {
	auth fleetmanager.Auth
}

func (f *tokenFetcher) FetchToken(_ awscredentials.Context) ([]byte, error) {
	token, err := f.auth.RetrieveIDToken()
	if err != nil {
		return nil, fmt.Errorf("retrieving token from token source: %w", err)
	}
	return []byte(token), nil
}
