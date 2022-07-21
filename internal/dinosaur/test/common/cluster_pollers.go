package common

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
)

const (
	clusterIDAssignmentTimeout = 2 * time.Minute
	clusterDeleteTimeout       = 15 * time.Minute
)

// WaitForClustersMatchCriteriaToBeGivenCount - Awaits for the number of clusters with an assigned cluster id to be exactly `count`
func WaitForClustersMatchCriteriaToBeGivenCount(db *db.ConnectionFactory, clusterService *services.ClusterService, clusterCriteria *services.FindClusterCriteria, count int) error {
	currentCount := -1
	return NewPollerBuilder(db).
		IntervalAndTimeout(defaultPollInterval, clusterIDAssignmentTimeout).
		RetryLogFunction(func(retry int, maxRetry int) string {
			if currentCount == -1 {
				return fmt.Sprintf("Waiting for cluster count to be %d", count)
			}
			return fmt.Sprintf("Waiting for cluster count to be %d (current: %d)", count, currentCount)
		}).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			clusters, svcErr := (*clusterService).FindAllClusters(*clusterCriteria)
			if svcErr != nil {
				return true, svcErr
			}
			currentCount = len(clusters)
			return currentCount == count, nil
		}).
		Build().Poll()
}

// WaitForClusterIDToBeAssigned - Awaits for clusterID to be assigned to the designed cluster
func WaitForClusterIDToBeAssigned(db *db.ConnectionFactory, clusterService *services.ClusterService, criteria *services.FindClusterCriteria) (string, error) {
	var clusterID string
	return clusterID, NewPollerBuilder(db).
		IntervalAndTimeout(defaultPollInterval, clusterIDAssignmentTimeout).
		RetryLogMessagef("Waiting for an ID to be assigned to the cluster (%+v)", criteria).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			foundCluster, svcErr := (*clusterService).FindCluster(*criteria)

			if svcErr != nil || foundCluster == nil {
				return true, fmt.Errorf("failed to find OSD cluster %s", svcErr)
			}
			clusterID = foundCluster.ClusterID
			return foundCluster.ClusterID != "", nil
		}).
		Build().Poll()
}

// WaitForClusterToBeDeleted - Awaits for the specified cluster to be deleted
func WaitForClusterToBeDeleted(db *db.ConnectionFactory, clusterService *services.ClusterService, clusterId string) error {
	return NewPollerBuilder(db).
		IntervalAndTimeout(defaultPollInterval, clusterDeleteTimeout).
		RetryLogMessagef("Waiting for cluster '%s' to be deleted", clusterId).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			clusterFromDb, findClusterByIdErr := (*clusterService).FindClusterByID(clusterId)
			if findClusterByIdErr != nil {
				return false, findClusterByIdErr
			}
			return clusterFromDb == nil, nil // cluster has been deleted
		}).
		Build().Poll()
}

// WaitForClusterStatus - Awaits for the cluster to reach the desired status
func WaitForClusterStatus(db *db.ConnectionFactory, clusterService *services.ClusterService, clusterId string, desiredStatus api.ClusterStatus) (cluster *api.Cluster, err error) {
	pollingInterval := defaultPollInterval
	if desiredStatus.String() != api.ClusterReady.String() {
		pollingInterval = 1 * time.Second
	}
	currentStatus := ""
	err = NewPollerBuilder(db).
		IntervalAndTimeout(pollingInterval, 120*time.Minute).
		DumpCluster(clusterId).
		RetryLogFunction(func(retry int, maxRetry int) string {
			if currentStatus == "" {
				return fmt.Sprintf("Waiting for cluster '%s' to reach status '%s'", clusterId, desiredStatus.String())
			}
			return fmt.Sprintf("Waiting for cluster '%s' to reach status '%s' (current status: '%s')", clusterId, desiredStatus.String(), currentStatus)
		}).
		OnRetry(func(attempt int, maxRetries int) (bool, error) {
			foundCluster, err := (*clusterService).FindClusterByID(clusterId)
			if err != nil {
				return true, err
			}
			if foundCluster == nil {
				return false, nil
			}
			cluster = foundCluster

			currentStatus = foundCluster.Status.String()

			if desiredStatus.CompareTo(api.ClusterDeprovisioning) < 0 && foundCluster.Status.CompareTo(api.ClusterDeprovisioning) >= 0 ||
				currentStatus == api.ClusterFailed.String() && desiredStatus.String() != api.ClusterFailed.String() {

				details := "N/A"

				if currentStatus == api.ClusterFailed.String() {
					// grab the logs
					if foundCluster, err = (*clusterService).CheckClusterStatus(foundCluster); err != nil {
						details = fmt.Sprintf("Error getting details: %s", err.Error())
					} else {
						details = cluster.StatusDetails
					}
				}

				return false, errors.Errorf("Waiting for cluster '%s' to reach status '%s' but reached status '%s' instead. Details: %s", clusterId, desiredStatus.String(), foundCluster.Status.String(), details)
			}

			return foundCluster.Status.CompareTo(desiredStatus) >= 0, nil
		}).Build().Poll()

	return cluster, err
}
