package reconciler

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *CentralReconciler) provisionRDSDatabase(remoteCentralNamespace string) (string, []byte, error) {
	cfg := &aws.Config{
		Region: aws.String(AWSRegion),
	}

	sess, err := session.NewSession(cfg)
	if err != nil {
		return "", nil, fmt.Errorf("unable to create session, %v", err)
	}

	clusterId := remoteCentralNamespace + "-db-cluster"
	instanceId := remoteCentralNamespace + "-db-instance"
	dbPassword := "not_random_pass_1234!" // TODO: generate password

	rdsClient := rds.New(sess)

	dbInstanceQuery := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(instanceId),
	}
	_, err = rdsClient.DescribeDBInstances(dbInstanceQuery)
	if err != nil {
		glog.Infof("Provisioning RDS database instance.")
		_, err = rdsClient.CreateDBCluster(newCreateCentralDBClusterInput(clusterId, dbPassword))
		if err != nil {
			return "", nil, fmt.Errorf("creating DB cluster: %v", err)
		}

		_, err = rdsClient.CreateDBInstance(newCreateCentralDBInstanceInput(clusterId, instanceId))
		if err != nil {
			// TODO: delete cluster
			return "", nil, fmt.Errorf("creating DB instance: %v", err)
		}
	}

	for {
		result, err := rdsClient.DescribeDBInstances(dbInstanceQuery)
		if err != nil {
			return "", nil, fmt.Errorf("retrieving DB instance state: %v", err)
		}

		if len(result.DBInstances) != 1 {
			return "", nil, fmt.Errorf("unexpected number of DB instances: %v", err)
		}

		dbInstanceStatus := *result.DBInstances[0].DBInstanceStatus
		if dbInstanceStatus == "available" {
			dbClusterQuery := &rds.DescribeDBClustersInput{
				DBClusterIdentifier: aws.String(clusterId),
			}

			clusterResult, err := rdsClient.DescribeDBClusters(dbClusterQuery)
			if err != nil {
				return "", nil, fmt.Errorf("retrieving DB cluster description: %v", err)
			}

			connectionString := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=require",
				*clusterResult.DBClusters[0].Endpoint, 5432, DBUser, "postgres")

			return connectionString, []byte(dbPassword), nil
		} else {
			fmt.Printf("Instance status: %s\n", dbInstanceStatus)
			time.Sleep(10 * time.Second)
		}
	}
}

func (r *CentralReconciler) ensureCentralDBSecretExists(ctx context.Context, remoteCentralNamespace string, password []byte) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: centralDbSecretName,
		},
	}
	err := r.client.Get(ctx, ctrlClient.ObjectKey{Namespace: remoteCentralNamespace, Name: centralDbSecretName}, secret)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      centralDbSecretName,
					Namespace: remoteCentralNamespace,
				},
				Data: map[string][]byte{"password": password},
			}

			err = r.client.Create(ctx, secret)
			if err != nil {
				return fmt.Errorf("creating Central DB secret: %v", err)
			}
			return nil
		}

		return fmt.Errorf("getting Central DB secret: %v", err)
	}

	return nil
}

func newCreateCentralDBClusterInput(clusterId string, dbPassword string) *rds.CreateDBClusterInput {
	return &rds.CreateDBClusterInput{
		DBClusterIdentifier: aws.String(clusterId),
		Engine:              aws.String(DBEngine),
		EngineVersion:       aws.String(DBEngineVersion),
		MasterUsername:      aws.String(DBUser),
		MasterUserPassword:  aws.String(dbPassword), // TODO: generate random password
		// TODO: the security group needs to be created during the data plane terraforming, and made known to
		// fleet-manager via a configuration parameter. I created this one in the AWS Console.
		VpcSecurityGroupIds: aws.StringSlice([]string{"sg-04dcc23a03646041c"}),
		ServerlessV2ScalingConfiguration: &rds.ServerlessV2ScalingConfiguration{
			MinCapacity: aws.Float64(0.5),
			MaxCapacity: aws.Float64(16),
		},
		BackupRetentionPeriod: aws.Int64(30),

		// TODO: The following are some extra parameters to consider
		// AvailabilityZones: // TODO: should we have multiple AZs?
		// DeletionProtection: // TODO: see if this useful
		// EnableCloudwatchLogsExports // TODO: enable
		// StorageEncrypted // TODO: enable
		// Tags // TODO: e.g. could add a tag that allows us to identify the associated Central
		// PreferredBackupWindow
		// PreferredMaintenanceWindow
	}
}

func newCreateCentralDBInstanceInput(clusterId, instanceId string) *rds.CreateDBInstanceInput {
	return &rds.CreateDBInstanceInput{
		DBInstanceClass:      aws.String(DBInstanceClass),
		DBClusterIdentifier:  aws.String(clusterId),
		DBInstanceIdentifier: aws.String(instanceId),
		Engine:               aws.String(DBEngine),
		PubliclyAccessible:   aws.Bool(true), // TODO: should be false
	}
}
