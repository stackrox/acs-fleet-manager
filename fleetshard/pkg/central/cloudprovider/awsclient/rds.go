// Package awsclient provides AWS-specific implementations of the interfaces in cloudprovider
package awsclient

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/cloudprovider"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/postgres"
)

const (
	dbAvailableStatus = "available"
	dbDeletingStatus  = "deleting"
	dbUser            = "rhacs_master"
	dbPrefix          = "rhacs-"
	dbInstanceSuffix  = "-db-instance"
	dbFailoverSuffix  = "-db-failover"
	dbClusterSuffix   = "-db-cluster"
	awsRetrySeconds   = 30

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
	instanceTypeTagKey      = "ACSInstanceType"
	regularInstaceTagValue  = "regular"
	testInstanceTagValue    = "test"

	// The Aurora Serverless v2 DB instance configuration in ACUs (Aurora Capacity Units)
	// 1 ACU = 1 vCPU + 2GB RAM
	dbMinCapacityACU = 0.5
	dbMaxCapacityACU = 16
)

// RDS is an AWS RDS client that provisions and deprovisions databases for ACS instances.
type RDS struct {
	dbSecurityGroup      string
	dbSubnetGroup        string
	performanceInsights  bool
	dataplaneClusterName string

	rdsClient *rds.RDS
}

// EnsureDBProvisioned is a blocking function that makes sure that an RDS database was provisioned for a Central
func (r *RDS) EnsureDBProvisioned(ctx context.Context, databaseID, masterPassword string, isTestInstance bool) error {
	clusterID := getClusterID(databaseID)
	if err := r.ensureDBClusterCreated(clusterID, masterPassword, isTestInstance); err != nil {
		return fmt.Errorf("ensuring DB cluster %s exists: %w", clusterID, err)
	}

	instanceID := getInstanceID(databaseID)
	if err := r.ensureDBInstanceCreated(instanceID, clusterID, isTestInstance); err != nil {
		return fmt.Errorf("ensuring DB instance %s exists in cluster %s: %w", instanceID, clusterID, err)
	}

	failoverID := getFailoverInstanceID(databaseID)
	if err := r.ensureDBInstanceCreated(failoverID, clusterID, isTestInstance); err != nil {
		return fmt.Errorf("ensuring failover DB instance %s exists in cluster %s: %w", failoverID, clusterID, err)
	}

	return r.waitForInstanceToBeAvailable(ctx, instanceID)
}

// EnsureDBDeprovisioned is a function that initiates the deprovisioning of the RDS database of a Central
// Unlike EnsureDBProvisioned, this function does not block until the DB is deprovisioned
func (r *RDS) EnsureDBDeprovisioned(databaseID string, skipFinalSnapshot bool) error {
	err := r.ensureInstanceDeleted(getInstanceID(databaseID))
	if err != nil {
		return err
	}

	err = r.ensureInstanceDeleted(getFailoverInstanceID(databaseID))
	if err != nil {
		return err
	}

	err = r.ensureClusterDeleted(getClusterID(databaseID), skipFinalSnapshot)
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

// GetAccountQuotas returns database-related service quotas for the AWS region on which
// the instance of fleetshard-sync runs
func (r *RDS) GetAccountQuotas(ctx context.Context) (cloudprovider.AccountQuotas, error) {
	accountAttributes, err := r.rdsClient.DescribeAccountAttributesWithContext(ctx, &rds.DescribeAccountAttributesInput{})
	if err != nil {
		return nil, fmt.Errorf("getting account quotas: %w", err)
	}

	neededQuotas := map[string]cloudprovider.AccountQuotaType{
		"DBInstances":            cloudprovider.DBInstances,
		"DBClusters":             cloudprovider.DBClusters,
		"ManualClusterSnapshots": cloudprovider.DBSnapshots,
	}

	accountQuotas := make(cloudprovider.AccountQuotas, len(neededQuotas))
	for _, quota := range accountAttributes.AccountQuotas {
		quotaType, ok := neededQuotas[*quota.AccountQuotaName]
		if ok {
			accountQuotas[quotaType] = cloudprovider.AccountQuotaValue{
				Used: *quota.Used,
				Max:  *quota.Max,
			}
		}
	}

	return accountQuotas, nil
}

func (r *RDS) ensureDBClusterCreated(clusterID, masterPassword string, isTestInstance bool) error {
	clusterExists, _, err := r.clusterStatus(clusterID)
	if err != nil {
		return fmt.Errorf("checking if DB cluster exists: %w", err)
	}
	if clusterExists {
		return nil
	}

	glog.Infof("Initiating provisioning of RDS database cluster %s.", clusterID)
	_, err = r.rdsClient.CreateDBCluster(newCreateCentralDBClusterInput(clusterID, masterPassword, r.dbSecurityGroup,
		r.dbSubnetGroup, r.dataplaneClusterName, isTestInstance))
	if err != nil {
		return fmt.Errorf("creating DB cluster: %w", err)
	}

	return nil
}

func (r *RDS) ensureDBInstanceCreated(instanceID string, clusterID string, isTestInstance bool) error {
	instanceExists, _, err := r.instanceStatus(instanceID)
	if err != nil {
		return fmt.Errorf("checking if DB instance exists: %w", err)
	}
	if instanceExists {
		return nil
	}

	glog.Infof("Initiating provisioning of RDS database instance %s.", instanceID)
	_, err = r.rdsClient.CreateDBInstance(newCreateCentralDBInstanceInput(clusterID, instanceID,
		r.dataplaneClusterName, r.performanceInsights, isTestInstance))
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

func (r *RDS) ensureClusterDeleted(clusterID string, skipFinalSnapshot bool) error {
	clusterExists, clusterStatus, err := r.clusterStatus(clusterID)
	if err != nil {
		return fmt.Errorf("getting DB cluster status: %w", err)
	}
	if !clusterExists {
		return nil
	}

	if clusterStatus != dbDeletingStatus {
		glog.Infof("Initiating deprovisioning of RDS database cluster %s.", clusterID)
		_, err := r.rdsClient.DeleteDBCluster(newDeleteCentralDBClusterInput(clusterID, skipFinalSnapshot))
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				// This assumes that if a final snapshot exists, a deletion for the RDS cluster was already triggered
				// and we can move on with deprovisioning,
				if awsErr.Code() == rds.ErrCodeDBClusterSnapshotAlreadyExistsFault {
					glog.Infof("Final DB backup is in progress for DB cluster: %s", clusterID)
					return nil
				}
			}
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
func NewRDSClient(config *config.Config) (*RDS, error) {
	rdsClient, err := newRdsClient()
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

func newCreateCentralDBClusterInput(clusterID, dbPassword, securityGroup, subnetGroup, dataplaneClusterName string, isTestInstance bool) *rds.CreateDBClusterInput {
	input := &rds.CreateDBClusterInput{
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
				Value: aws.String(dataplaneClusterName),
			},
			{
				Key:   aws.String(instanceTypeTagKey),
				Value: aws.String(getInstanceType(isTestInstance)),
			},
		},
	}

	// do not export DB logs of internal instances (e.g. Probes)
	if !isTestInstance {
		input.EnableCloudwatchLogsExports = aws.StringSlice([]string{"postgresql"})
	}

	return input
}

func newCreateCentralDBInstanceInput(clusterID, instanceID, dataplaneClusterName string, performanceInsights bool, isTestInstance bool) *rds.CreateDBInstanceInput {
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
				Value: aws.String(dataplaneClusterName),
			},
			{
				Key:   aws.String(instanceTypeTagKey),
				Value: aws.String(getInstanceType(isTestInstance)),
			},
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

func newRdsClient() (*rds.RDS, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, fmt.Errorf("unable to create session for RDS client: %w", err)
	}

	return rds.New(sess), nil
}

func getFinalSnapshotID(clusterID string) *string {
	return aws.String(fmt.Sprintf("%s-%s", clusterID, "final"))
}

func getInstanceType(isTestInstance bool) string {
	if isTestInstance {
		return testInstanceTagValue
	}
	return regularInstaceTagValue
}
