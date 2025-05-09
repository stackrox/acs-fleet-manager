package awsclient

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/google/uuid"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/cloudprovider"
	"github.com/stackrox/rox/pkg/random"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/rand"
)

const awsTimeoutMinutes = 30

func newTestRDS() (*RDS, error) {
	rdsClient, err := newTestRDSClient()
	if err != nil {
		return nil, fmt.Errorf("unable to create RDS client: %w", err)
	}

	return &RDS{
		rdsClient:       rdsClient,
		dbSecurityGroup: os.Getenv("MANAGED_DB_SECURITY_GROUP"),
		dbSubnetGroup:   os.Getenv("MANAGED_DB_SUBNET_GROUP"),
	}, nil
}

func newTestRDSClient() (*rds.RDS, error) {
	cfg := &aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	}

	sess, err := session.NewSession(cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create session, %w", err)
	}

	return rds.New(sess), nil
}

func waitForClusterToBeDeleted(ctx context.Context, rdsClient *RDS, clusterID string) (bool, error) {
	for {
		clusterExists, _, err := rdsClient.clusterStatus(clusterID)
		if err != nil {
			return false, err
		}

		if !clusterExists {
			return true, nil
		}

		ticker := time.NewTicker(awsRetrySeconds * time.Second)
		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return false, fmt.Errorf("waiting for RDS cluster to be deleted: %w", ctx.Err())
		}
	}
}

func waitForFinalSnapshotToExist(ctx context.Context, rdsClient *RDS, clusterID string) (bool, string, error) {

	ticker := time.NewTicker(awsRetrySeconds * time.Second)
	for {
		select {
		case <-ticker.C:
			snapshotOut, err := rdsClient.rdsClient.DescribeDBClusterSnapshots(&rds.DescribeDBClusterSnapshotsInput{
				DBClusterIdentifier: &clusterID,
			})

			if err != nil {
				if awsErr, ok := err.(awserr.Error); ok {
					if awsErr.Code() != rds.ErrCodeDBClusterSnapshotNotFoundFault {
						return false, "", err
					}

					continue
				}
			}

			if snapshotOut != nil && len(snapshotOut.DBClusterSnapshots) == 1 {
				return true, *snapshotOut.DBClusterSnapshots[0].DBClusterSnapshotIdentifier, nil
			}

		case <-ctx.Done():
			return false, "", fmt.Errorf("waiting for final DB snapshot: %w", ctx.Err())
		}

	}

}

func TestRDSProvisioning(t *testing.T) {
	if os.Getenv("RUN_AWS_INTEGRATION") != "true" {
		t.Skip("Skip RDS tests. Set RUN_AWS_INTEGRATION=true env variable to enable RDS tests.")
	}

	rdsClient, err := newTestRDS()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.TODO(), awsTimeoutMinutes*time.Minute)
	defer cancel()

	dbID := "test-" + uuid.New().String()
	dbMasterPassword, err := random.GenerateString(25, random.AlphanumericCharacters)
	require.NoError(t, err)

	clusterID := getClusterID(dbID)
	instanceID := getInstanceID(dbID)
	failoverID := getFailoverInstanceID(dbID)

	clusterExists, _, err := rdsClient.clusterStatus(clusterID)
	require.NoError(t, err)
	require.False(t, clusterExists)

	instanceExists, _, err := rdsClient.clusterStatus(instanceID)
	require.NoError(t, err)
	require.False(t, instanceExists)

	failoverExists, _, err := rdsClient.clusterStatus(failoverID)
	require.NoError(t, err)
	require.False(t, failoverExists)

	err = rdsClient.EnsureDBProvisioned(ctx, dbID, dbID, dbMasterPassword, false)
	defer func() {
		// clean-up AWS resources in case the test fails
		deleteErr := rdsClient.EnsureDBDeprovisioned(dbID, false)
		assert.NoError(t, deleteErr)
	}()
	require.NoError(t, err)

	_, err = rdsClient.GetDBConnection(dbID)
	assert.NoError(t, err)

	clusterExists, clusterStatus, err := rdsClient.clusterStatus(clusterID)
	require.NoError(t, err)
	require.True(t, clusterExists)
	assert.Equal(t, clusterStatus, dbAvailableStatus)

	instanceExists, instanceStatus, err := rdsClient.instanceStatus(instanceID)
	require.NoError(t, err)
	require.True(t, instanceExists)
	assert.Equal(t, instanceStatus, dbAvailableStatus)

	failoverExists, _, err = rdsClient.instanceStatus(failoverID)
	require.NoError(t, err)
	require.True(t, failoverExists)

	err = rdsClient.EnsureDBDeprovisioned(dbID, false)
	assert.NoError(t, err)

	deleteCtx, deleteCancel := context.WithTimeout(context.TODO(), awsTimeoutMinutes*time.Minute)
	defer deleteCancel()

	clusterDeleted, err := waitForClusterToBeDeleted(deleteCtx, rdsClient, clusterID)
	require.NoError(t, err)
	assert.True(t, clusterDeleted)

	snapshotExists, snapshotID, err := waitForFinalSnapshotToExist(deleteCtx, rdsClient, clusterID)

	if snapshotExists {
		defer func() {
			_, err := rdsClient.rdsClient.DeleteDBClusterSnapshot(
				&rds.DeleteDBClusterSnapshotInput{DBClusterSnapshotIdentifier: &snapshotID},
			)

			assert.NoError(t, err)
		}()
	}

	require.NoError(t, err)
	require.True(t, snapshotExists)
}

func TestGetDBConnection(t *testing.T) {
	if os.Getenv("RUN_AWS_INTEGRATION") != "true" {
		t.Skip("Skip RDS tests. Set RUN_AWS_INTEGRATION=true env variable to enable RDS tests.")
	}

	rdsClient, err := newTestRDS()
	require.NoError(t, err)

	_, err = rdsClient.GetDBConnection("test-" + uuid.New().String())
	var awsErr awserr.Error
	require.ErrorAs(t, err, &awsErr)
	assert.Equal(t, awsErr.Code(), rds.ErrCodeDBClusterNotFoundFault)
	require.ErrorIs(t, err, cloudprovider.ErrDBNotFound)
}

func TestGetAccountQuotas(t *testing.T) {
	if os.Getenv("RUN_AWS_INTEGRATION") != "true" {
		t.Skip("Skip RDS tests. Set RUN_AWS_INTEGRATION=true env variable to enable RDS tests.")
	}

	rdsClient, err := newTestRDS()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	accountQuotas, err := rdsClient.GetAccountQuotas(ctx)
	require.NoError(t, err)

	expectedQuotas := [...]cloudprovider.AccountQuotaType{cloudprovider.DBClusters, cloudprovider.DBInstances, cloudprovider.DBSnapshots}
	for _, quota := range expectedQuotas {
		quotaValue, found := accountQuotas[quota]
		require.True(t, found)
		var minQuotaValue int64
		assert.GreaterOrEqual(t, quotaValue.Used, minQuotaValue)
		assert.GreaterOrEqual(t, quotaValue.Max, minQuotaValue)
	}
}

func TestRestoreIfFinalSnapshotExists(t *testing.T) {
	mockRDSClient := RDSClientMock{}
	tenantID := "veryrandomid"
	clusterID := "rhacs-veryrandomid"
	finalSnapshotID := getFinalSnapshotID(clusterID)

	// Mocking describe cluster to return not found, which triggers creation
	mockRDSClient.DescribeDBClustersFunc = func(describeDBClustersInput *rds.DescribeDBClustersInput) (*rds.DescribeDBClustersOutput, error) {
		return nil, awserr.New(rds.ErrCodeDBClusterNotFoundFault, "db cluster not found", errors.New("db cluster not found"))
	}

	// Mocking describe snapshots to return a valid final snapshot and catch input
	var describeSnapshotInput *rds.DescribeDBClusterSnapshotsInput
	mockRDSClient.DescribeDBClusterSnapshotsFunc = func(describeDBClusterSnapshotsInput *rds.DescribeDBClusterSnapshotsInput) (*rds.DescribeDBClusterSnapshotsOutput, error) {
		describeSnapshotInput = describeDBClusterSnapshotsInput
		return &rds.DescribeDBClusterSnapshotsOutput{
			DBClusterSnapshots: []*rds.DBClusterSnapshot{
				// using multiple snapshots to make sure we find the final in the mid of available snapshots
				{DBClusterSnapshotIdentifier: randomNonFinalSnapshotsID(clusterID)},
				{DBClusterSnapshotIdentifier: randomNonFinalSnapshotsID(clusterID)},
				{DBClusterSnapshotIdentifier: randomNonFinalSnapshotsID(clusterID)},
				{DBClusterSnapshotIdentifier: finalSnapshotID},
				{DBClusterSnapshotIdentifier: randomNonFinalSnapshotsID(clusterID)},
				{DBClusterSnapshotIdentifier: randomNonFinalSnapshotsID(clusterID)},
			},
		}, nil
	}

	// Mocking cluster restore function to catch input
	var restoreInput *rds.RestoreDBClusterFromSnapshotInput
	mockRDSClient.RestoreDBClusterFromSnapshotFunc = func(restoreDBClusterFromSnapshotInput *rds.RestoreDBClusterFromSnapshotInput) (*rds.RestoreDBClusterFromSnapshotOutput, error) {
		restoreInput = restoreDBClusterFromSnapshotInput
		return &rds.RestoreDBClusterFromSnapshotOutput{}, nil
	}

	// Mocking create function to make sure it was not called
	var createCalled bool
	mockRDSClient.CreateDBClusterFunc = func(createDBClusterInput *rds.CreateDBClusterInput) (*rds.CreateDBClusterOutput, error) {
		createCalled = true
		return nil, nil
	}
	// create function should not be called for restore operations

	rds := RDS{
		rdsClient: &mockRDSClient,
	}

	err := rds.ensureDBClusterCreated(clusterID, tenantID, "testpassword1234", false)

	require.NoError(t, err)
	require.NotNil(t, describeSnapshotInput)
	require.NotNil(t, restoreInput)
	assert.Equal(t, *describeSnapshotInput.DBClusterIdentifier, clusterID)
	assert.Equal(t, *restoreInput.SnapshotIdentifier, *finalSnapshotID)
	assert.False(t, createCalled)
}

func randomNonFinalSnapshotsID(clusterID string) *string {
	return aws.String(fmt.Sprintf("%s-%s", clusterID, rand.String(20)))
}
