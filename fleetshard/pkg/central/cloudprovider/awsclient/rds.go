// Package awsclient provides AWS-specific implementations of the interfaces in cloudprovider
package awsclient

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/cloudprovider"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/postgres"
	"k8s.io/apimachinery/pkg/util/rand"
)

// RDSClient is a wrapper around the rdsiface.RDSAPI to generate a mock
//
//go:generate moq -out rds_client_moq.go . RDSClient
type RDSClient interface {
	DescribeDBClusterSnapshots(ctx context.Context, params *rds.DescribeDBClusterSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClusterSnapshotsOutput, error)
	DescribeAccountAttributes(ctx context.Context, params *rds.DescribeAccountAttributesInput, optFns ...func(*rds.Options)) (*rds.DescribeAccountAttributesOutput, error)
	DescribeDBInstances(ctx context.Context, params *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error)
	DescribeDBClusters(ctx context.Context, params *rds.DescribeDBClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClustersOutput, error)

	CreateDBCluster(ctx context.Context, params *rds.CreateDBClusterInput, optFns ...func(*rds.Options)) (*rds.CreateDBClusterOutput, error)
	CreateDBInstance(ctx context.Context, params *rds.CreateDBInstanceInput, optFns ...func(*rds.Options)) (*rds.CreateDBInstanceOutput, error)

	RestoreDBClusterFromSnapshot(ctx context.Context, params *rds.RestoreDBClusterFromSnapshotInput, optFns ...func(*rds.Options)) (*rds.RestoreDBClusterFromSnapshotOutput, error)

	DeleteDBInstance(ctx context.Context, params *rds.DeleteDBInstanceInput, optFns ...func(*rds.Options)) (*rds.DeleteDBInstanceOutput, error)
	DeleteDBCluster(ctx context.Context, params *rds.DeleteDBClusterInput, optFns ...func(*rds.Options)) (*rds.DeleteDBClusterOutput, error)
	DeleteDBClusterSnapshot(ctx context.Context, params *rds.DeleteDBClusterSnapshotInput, optFns ...func(*rds.Options)) (*rds.DeleteDBClusterSnapshotOutput, error)
}

const (
	dbAvailableStatus = "available"
	dbDeletingStatus  = "deleting"
	dbBackingUpStatus = "backing-up"
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

	instanceTypeTagKey     = "ACSInstanceType"
	acsInstanceIDKey       = "ACSInstanceID"
	regularInstaceTagValue = "regular"
	testInstanceTagValue   = "test"

	// The Aurora Serverless v2 DB instance configuration in ACUs (Aurora Capacity Units)
	// 1 ACU = 1 vCPU + 2GB RAM
	dbMinCapacityACU = 0.5
	dbMaxCapacityACU = 16
)

// RDS is an AWS RDS client that provisions and deprovisions databases for ACS instances.
type RDS struct {
	dbSecurityGroup     string
	dbSubnetGroup       string
	performanceInsights bool
	sharedTags          []config.ManagedDBTag
	rdsClient           RDSClient
}

// EnsureDBProvisioned is a blocking function that makes sure that an RDS database was provisioned for a Central
func (r *RDS) EnsureDBProvisioned(ctx context.Context, databaseID, acsInstanceID, masterPassword string, isTestInstance bool) error {
	clusterID := getClusterID(databaseID)
	if err := r.ensureDBClusterCreated(clusterID, acsInstanceID, masterPassword, isTestInstance); err != nil {
		return fmt.Errorf("ensuring DB cluster %s exists: %w", clusterID, err)
	}

	instanceID := getInstanceID(databaseID)
	if err := r.ensureDBInstanceCreated(instanceID, clusterID, acsInstanceID, isTestInstance); err != nil {
		return fmt.Errorf("ensuring DB instance %s exists in cluster %s: %w", instanceID, clusterID, err)
	}

	failoverID := getFailoverInstanceID(databaseID)
	if err := r.ensureDBInstanceCreated(failoverID, clusterID, acsInstanceID, isTestInstance); err != nil {
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
		var awsErr *types.DBClusterNotFoundFault
		if errors.As(err, &awsErr) {
			err = errors.Join(cloudprovider.ErrDBNotFound, err)
		}

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
	accountAttributes, err := r.rdsClient.DescribeAccountAttributes(ctx, &rds.DescribeAccountAttributesInput{})
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

func (r *RDS) ensureDBClusterCreated(clusterID, acsInstanceID, masterPassword string, isTestInstance bool) error {
	clusterExists, _, err := r.clusterStatus(clusterID)
	if err != nil {
		return fmt.Errorf("checking if DB cluster exists: %w", err)
	}
	if clusterExists {
		return nil
	}

	finalSnapshotID, err := r.getFinalSnapshotIDIfExists(clusterID)
	if err != nil {
		return err
	}

	glog.Infof("Initiating provisioning of RDS database cluster %s.", clusterID)

	input := &createCentralDBClusterInput{
		clusterID:      clusterID,
		acsInstanceID:  acsInstanceID,
		dbPassword:     masterPassword, // pragma: allowlist secret
		securityGroup:  r.dbSecurityGroup,
		subnetGroup:    r.dbSubnetGroup,
		isTestInstance: isTestInstance,
	}

	rdsCreateDBClusterInput := r.newCreateCentralDBClusterInput(input)

	if finalSnapshotID != "" {
		glog.Infof("Restoring DB cluster: %s from snasphot: %s", clusterID, finalSnapshotID)
		return r.restoreDBClusterFromSnapshot(finalSnapshotID, rdsCreateDBClusterInput)
	}

	return r.createDBCluster(rdsCreateDBClusterInput)
}

func (r *RDS) restoreDBClusterFromSnapshot(snapshotID string, clusterInput *rds.CreateDBClusterInput) error {
	_, err := r.rdsClient.RestoreDBClusterFromSnapshot(context.TODO(), newRestoreCentralDBClusterInput(snapshotID, clusterInput))
	if err != nil {
		return fmt.Errorf("restoring DB cluster: %w", err)
	}

	return nil
}

func (r *RDS) createDBCluster(clusterInput *rds.CreateDBClusterInput) error {
	_, err := r.rdsClient.CreateDBCluster(context.TODO(), clusterInput)
	if err != nil {
		return fmt.Errorf("creating DB cluster: %w", err)
	}

	return nil
}

func (r *RDS) getFinalSnapshotIDIfExists(clusterID string) (string, error) {
	snapshotsOut, err := r.rdsClient.DescribeDBClusterSnapshots(context.TODO(), &rds.DescribeDBClusterSnapshotsInput{
		DBClusterIdentifier: &clusterID,
	})

	if err != nil {
		return "", fmt.Errorf("checking if final snapshot for clusterID: %s exists: %w", clusterID, err)
	}

	var mostRecentSnapshotID string
	var mostRecentSnapshotTime *time.Time
	for _, snapshot := range snapshotsOut.DBClusterSnapshots {
		if !strings.Contains(*snapshot.DBClusterSnapshotIdentifier, "final") {
			continue
		}

		if mostRecentSnapshotTime == nil || mostRecentSnapshotTime.Before(*snapshot.SnapshotCreateTime) {
			mostRecentSnapshotID = *snapshot.DBClusterSnapshotIdentifier
			mostRecentSnapshotTime = snapshot.SnapshotCreateTime
		}
	}

	return mostRecentSnapshotID, nil
}

func (r *RDS) ensureDBInstanceCreated(instanceID, clusterID, acsInstanceID string, isTestInstance bool) error {
	instanceExists, _, err := r.instanceStatus(instanceID)
	if err != nil {
		return fmt.Errorf("checking if DB instance exists: %w", err)
	}
	if instanceExists {
		return nil
	}

	glog.Infof("Initiating provisioning of RDS database instance %s.", instanceID)
	input := &createCentralDBInstanceInput{
		clusterID:           clusterID,
		instanceID:          instanceID,
		acsInstanceID:       acsInstanceID,
		performanceInsights: r.performanceInsights,
		isTestInstance:      isTestInstance,
	}
	_, err = r.rdsClient.CreateDBInstance(context.TODO(), r.newCreateCentralDBInstanceInput(input))
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
		_, err := r.rdsClient.DeleteDBInstance(context.TODO(), newDeleteCentralDBInstanceInput(instanceID, true))
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

	if clusterStatus == dbBackingUpStatus {
		return cloudprovider.ErrDBBackupInProgress
	}

	if clusterStatus != dbDeletingStatus {
		glog.Infof("Initiating deprovisioning of RDS database cluster %s.", clusterID)
		_, err := r.rdsClient.DeleteDBCluster(context.TODO(), newDeleteCentralDBClusterInput(clusterID, skipFinalSnapshot))
		if err != nil {
			var alreadyExists *types.DBClusterSnapshotAlreadyExistsFault
			if errors.As(err, &alreadyExists) {
				// This assumes that if a final snapshot exists, a deletion for the RDS cluster was already triggered
				// and we can move on with deprovisioning,
				glog.Infof("Final DB backup is in progress for DB cluster: %s", clusterID)
				return nil
			}
			return fmt.Errorf("deleting DB cluster: %w", err)

		}
	}

	return nil
}

func (r *RDS) clusterStatus(clusterID string) (bool, string, error) {
	dbCluster, err := r.describeDBCluster(clusterID)
	if err != nil {
		var clusterNotFound *types.DBClusterNotFoundFault
		if errors.As(err, &clusterNotFound) {
			return false, "", nil
		}
		return false, "", err
	}

	return true, *dbCluster.Status, nil
}

func (r *RDS) instanceStatus(instanceID string) (bool, string, error) {
	dbInstance, err := r.describeDBInstance(instanceID)
	if err != nil {
		var instanceNotFound *types.DBInstanceNotFoundFault
		if errors.As(err, &instanceNotFound) {
			return false, "", nil
		}
		return false, "", err
	}

	return true, *dbInstance.DBInstanceStatus, nil
}

func (r *RDS) describeDBInstance(instanceID string) (*types.DBInstance, error) {
	result, err := r.rdsClient.DescribeDBInstances(context.TODO(),
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

	return &result.DBInstances[0], nil
}

func (r *RDS) describeDBCluster(clusterID string) (*types.DBCluster, error) {
	result, err := r.rdsClient.DescribeDBClusters(context.TODO(), &rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(clusterID),
	})
	if err != nil {
		return nil, fmt.Errorf("retrieving DB cluster description: %w", err)
	}

	if len(result.DBClusters) != 1 {
		// this should never happen (DescribeDBClusters should return either 1 cluster, or ErrCodeDBClusterNotFoundFault)
		return nil, fmt.Errorf("unexpected number of DB clusters: %d", len(result.DBClusters))
	}

	return &result.DBClusters[0], nil
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
		rdsClient:           rdsClient,
		dbSecurityGroup:     config.ManagedDB.SecurityGroup,
		dbSubnetGroup:       config.ManagedDB.SubnetGroup,
		performanceInsights: config.ManagedDB.PerformanceInsights,
		sharedTags:          config.ManagedDB.SharedTags,
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

type createCentralDBClusterInput struct {
	clusterID      string
	acsInstanceID  string
	dbPassword     string
	securityGroup  string
	subnetGroup    string
	isTestInstance bool
}

func (r *RDS) newCreateCentralDBClusterInput(input *createCentralDBClusterInput) *rds.CreateDBClusterInput {
	awsInput := &rds.CreateDBClusterInput{
		DBClusterIdentifier: aws.String(input.clusterID),
		Engine:              aws.String(dbEngine),
		EngineVersion:       aws.String(dbEngineVersion),
		MasterUsername:      aws.String(dbUser),
		MasterUserPassword:  aws.String(input.dbPassword),
		VpcSecurityGroupIds: []string{input.securityGroup},
		DBSubnetGroupName:   aws.String(input.subnetGroup),
		ServerlessV2ScalingConfiguration: &types.ServerlessV2ScalingConfiguration{
			MinCapacity: aws.Float64(dbMinCapacityACU),
			MaxCapacity: aws.Float64(dbMaxCapacityACU),
		},
		BackupRetentionPeriod: aws.Int32(dbBackupRetentionPeriod),
		StorageEncrypted:      aws.Bool(true),
		Tags:                  r.getDesiredTags(input.acsInstanceID, input.isTestInstance),
	}

	// do not export DB logs of internal instances (e.g. Probes)
	if !input.isTestInstance {
		awsInput.EnableCloudwatchLogsExports = []string{"postgresql"}
	}

	return awsInput
}

func (r *RDS) getDesiredTags(acsInstanceID string, isTestInstance bool) []types.Tag {
	tags := make([]types.Tag, 0, len(r.sharedTags)+2)
	for _, tag := range r.sharedTags {
		tags = append(tags, types.Tag{
			Key:   aws.String(tag.Key),
			Value: aws.String(tag.Value),
		})
	}

	tags = append(tags, types.Tag{
		Key:   aws.String(instanceTypeTagKey),
		Value: aws.String(getInstanceType(isTestInstance)),
	}, types.Tag{
		Key:   aws.String(acsInstanceIDKey),
		Value: aws.String(acsInstanceID),
	})

	return tags
}

func newRestoreCentralDBClusterInput(snapshotID string, input *rds.CreateDBClusterInput) *rds.RestoreDBClusterFromSnapshotInput {
	restoreInput := &rds.RestoreDBClusterFromSnapshotInput{
		DBClusterIdentifier:              input.DBClusterIdentifier,
		Engine:                           input.Engine,
		EngineVersion:                    input.EngineVersion,
		VpcSecurityGroupIds:              input.VpcSecurityGroupIds,
		PubliclyAccessible:               input.PubliclyAccessible,
		DBSubnetGroupName:                input.DBSubnetGroupName,
		ServerlessV2ScalingConfiguration: input.ServerlessV2ScalingConfiguration,
		Tags:                             input.Tags,
		SnapshotIdentifier:               &snapshotID,
		EnableCloudwatchLogsExports:      input.EnableCloudwatchLogsExports,
	}

	return restoreInput
}

type createCentralDBInstanceInput struct {
	clusterID           string
	instanceID          string
	acsInstanceID       string
	performanceInsights bool
	isTestInstance      bool
}

func (r *RDS) newCreateCentralDBInstanceInput(input *createCentralDBInstanceInput) *rds.CreateDBInstanceInput {
	return &rds.CreateDBInstanceInput{
		DBInstanceClass:           aws.String(dbInstanceClass),
		DBClusterIdentifier:       aws.String(input.clusterID),
		DBInstanceIdentifier:      aws.String(input.instanceID),
		Engine:                    aws.String(dbEngine),
		PubliclyAccessible:        aws.Bool(false),
		EnablePerformanceInsights: aws.Bool(input.performanceInsights),
		PromotionTier:             aws.Int32(dbInstancePromotionTier),
		CACertificateIdentifier:   aws.String(dbCACertificateType),
		AutoMinorVersionUpgrade:   aws.Bool(dbAutoVersionUpgrade),

		Tags: r.getDesiredTags(input.acsInstanceID, input.isTestInstance),
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

func newRdsClient() (*rds.Client, error) {
	cfg, err := awsConfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS SDK config: %w", err)
	}

	return rds.NewFromConfig(cfg), nil
}

func getFinalSnapshotID(clusterID string) *string {
	return aws.String(fmt.Sprintf("%s-%s-%s", clusterID, rand.String(10), "final"))
}

func getInstanceType(isTestInstance bool) string {
	if isTestInstance {
		return testInstanceTagValue
	}
	return regularInstaceTagValue
}
