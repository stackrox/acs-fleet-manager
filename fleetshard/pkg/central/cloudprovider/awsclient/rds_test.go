package awsclient

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/google/uuid"
	"github.com/stackrox/rox/pkg/random"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRDS() (*RDS, error) {
	rdsClient, err := newTestRDSClient()
	if err != nil {
		return nil, fmt.Errorf("unable to create RDS client: %w", err)
	}

	return &RDS{
		rdsClient:       rdsClient,
		dbSecurityGroup: os.Getenv("AWS_RDS_SECURITY_GROUP_ID"),
		dbSubnetGroup:   os.Getenv("AWS_RDS_DB_SUBNET_GROUP_NAME"),
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

func TestRDSProvisioning(t *testing.T) {
	if os.Getenv("RUN_RDS_TESTS") != "true" {
		t.Skip("Skip RDS tests. Set RUN_RDS_TESTS=true env variable to enable RDS tests.")
	}

	rdsClient, err := newTestRDS()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.TODO(), 15*time.Minute)
	defer cancel()

	dbID := "test-" + uuid.New().String()
	dbMasterPassword, err := random.GenerateString(25, random.AlphanumericCharacters)
	require.NoError(t, err)

	clusterID := getClusterID(dbID)
	instanceID := getInstanceID(dbID)

	clusterExists, err := rdsClient.clusterExists(clusterID)
	require.NoError(t, err)
	require.False(t, clusterExists)

	instanceExists, err := rdsClient.instanceExists(instanceID)
	require.NoError(t, err)
	require.False(t, instanceExists)

	_, err = rdsClient.EnsureDBProvisioned(ctx, dbID, dbMasterPassword)
	assert.NoError(t, err)

	clusterExists, err = rdsClient.clusterExists(clusterID)
	require.NoError(t, err)
	require.True(t, clusterExists)

	instanceExists, err = rdsClient.instanceExists(instanceID)
	require.NoError(t, err)
	require.True(t, instanceExists)

	clusterStatus, err := rdsClient.clusterStatus(clusterID)
	require.NoError(t, err)
	assert.Equal(t, clusterStatus, dbAvailableStatus)

	instanceStatus, err := rdsClient.instanceStatus(instanceID)
	require.NoError(t, err)
	assert.Equal(t, instanceStatus, dbAvailableStatus)

	deletionStarted, err := rdsClient.EnsureDBDeprovisioned(dbID)
	assert.NoError(t, err)
	assert.True(t, deletionStarted)

	// deletion takes a few minutes, so the cluster and instance should still exist
	clusterExists, err = rdsClient.clusterExists(clusterID)
	require.NoError(t, err)
	assert.True(t, clusterExists)

	instanceExists, err = rdsClient.instanceExists(instanceID)
	require.NoError(t, err)
	assert.True(t, instanceExists)

	clusterStatus, err = rdsClient.clusterStatus(clusterID)
	require.NoError(t, err)
	assert.Equal(t, clusterStatus, dbDeletingStatus)

	instanceStatus, err = rdsClient.instanceStatus(instanceID)
	require.NoError(t, err)
	assert.Equal(t, instanceStatus, dbDeletingStatus)
}
