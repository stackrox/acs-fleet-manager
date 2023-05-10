package awsclient

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/google/uuid"
	"github.com/stackrox/rox/pkg/random"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func waitForFinalSnapshotToExist(ctx context.Context, rdsClient *RDS, clusterID string) (bool, error) {

	ticker := time.NewTicker(awsRetrySeconds * time.Second)
	for {
		select {
		case <-ticker.C:
			snapshotOut, err := rdsClient.rdsClient.DescribeDBClusterSnapshots(&rds.DescribeDBClusterSnapshotsInput{
				DBClusterSnapshotIdentifier: getFinalSnapshotID(clusterID),
			})

			if err != nil {
				if awsErr, ok := err.(awserr.Error); ok {
					if awsErr.Code() != rds.ErrCodeDBClusterSnapshotNotFoundFault {
						return false, err
					}

					continue
				}
			}

			if snapshotOut != nil {
				return len(snapshotOut.DBClusterSnapshots) == 1, nil
			}
		case <-ctx.Done():
			return false, fmt.Errorf("waiting for final DB snapshot: %w", ctx.Err())
		}

	}

}

func TestRDSProvisioning(t *testing.T) {
	if os.Getenv("RUN_RDS_TESTS") != "true" {
		t.Skip("Skip RDS tests. Set RUN_RDS_TESTS=true env variable to enable RDS tests.")
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

	err = rdsClient.EnsureDBProvisioned(ctx, dbID, dbMasterPassword)
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

	// Always attempt to delete the final snapshot if it exists
	defer func() {
		_, err := rdsClient.rdsClient.DeleteDBClusterSnapshot(
			&rds.DeleteDBClusterSnapshotInput{DBClusterSnapshotIdentifier: getFinalSnapshotID(clusterID)},
		)

		assert.NoError(t, err)
	}()

	snapshotExists, err := waitForFinalSnapshotToExist(deleteCtx, rdsClient, clusterID)
	require.NoError(t, err)
	require.True(t, snapshotExists)
}

func TestGetDBConnection(t *testing.T) {
	if os.Getenv("RUN_RDS_TESTS") != "true" {
		t.Skip("Skip RDS tests. Set RUN_RDS_TESTS=true env variable to enable RDS tests.")
	}

	rdsClient, err := newTestRDS()
	require.NoError(t, err)

	_, err = rdsClient.GetDBConnection("test-" + uuid.New().String())
	var awsErr awserr.Error
	require.ErrorAs(t, err, &awsErr)
	assert.Equal(t, awsErr.Code(), rds.ErrCodeDBClusterNotFoundFault)
}
