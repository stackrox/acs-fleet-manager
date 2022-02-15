package common

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/server"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"

	. "github.com/onsi/gomega"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	ocmErrors "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/test/mocks"
)

const (
	testClusterPath = "test_cluster.json"
)

// GetRunningOsdClusterID - is used by tests to get a ClusterID value of an existing OSD cluster in a 'ready' state.
// If executed against real OCM client, content of internal/dinosaur/test/integration/test_cluster.json file (if present) is read
// to determine if there is a cluster running and new entry is added to the clusters table.
// If no such file is present, new cluster is created and its clusterID is retrieved.
func GetRunningOsdClusterID(h *test.Helper, t *testing.T) (string, *ocmErrors.ServiceError) {
	return GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterReady)
}

func GetOSDClusterIDAndWaitForStatus(h *test.Helper, t *testing.T, expectedStatus api.ClusterStatus) (string, *ocmErrors.ServiceError) {
	return GetOSDClusterID(h, t, &expectedStatus)
}

// GetOSDClusterID - is used by tests to get a ClusterID value of an existing OSD cluster.
// If executed against real OCM client, content of internal/dinosaur/test/integration/test_cluster.json file (if present) is read
// to determine if there is a cluster running and new entry is added to the clusters table.
// If no such file is present, new cluster is created and its clusterID is retrieved.
// If expectedStatus is not nil then it will wait until expectedStatus is reached.
func GetOSDClusterID(h *test.Helper, t *testing.T, expectedStatus *api.ClusterStatus) (string, *ocmErrors.ServiceError) {
	var foundCluster *api.Cluster
	var err *ocmErrors.ServiceError
	var clusterService services.ClusterService
	var ocmConfig ocm.OCMConfig
	h.Env.MustResolveAll(&clusterService)

	// get cluster details from persisted cluster file
	if fileExists(testClusterPath, t) {
		clusterID, _ := readClusterDetailsFromFile(h, t)
		foundCluster, err = clusterService.FindClusterByID(clusterID)
		if err != nil {
			return "", err
		}
	}

	// get cluster details from database when running against an emulated server and the cluster in cluster file doesn't exist
	if foundCluster == nil && ocmConfig.MockMode == ocm.MockModeEmulateServer {
		foundCluster, err = findFirstValidCluster(h)
		if err != nil {
			return "", err
		}
	}

	// No existing cluster found
	if foundCluster == nil {
		// create a new cluster
		_, err := clusterService.Create(&api.Cluster{
			CloudProvider: mocks.MockCluster.CloudProvider().ID(),
			Region:        mocks.MockCluster.Region().ID(),
			MultiAZ:       mocks.MockMultiAZ,
		})
		if err != nil {
			return "", err
		}

		// need to wait for it to be assigned
		waitErr := NewPollerBuilder(h.DBFactory()).
			IntervalAndTimeout(1*time.Second, 5*time.Minute).
			RetryLogMessage("Waiting for ID to be assigned to the new cluster").
			OnRetry(func(attempt int, maxRetries int) (bool, error) {
				c, findErr := findFirstValidCluster(h)
				if findErr != nil {
					return false, findErr
				}
				foundCluster = c
				return foundCluster != nil && foundCluster.ClusterID != "", nil
			}).Build().Poll()

		if waitErr != nil {
			return "", ocmErrors.GeneralError("error to find a valid cluster: %v", waitErr)
		}

		// Only persist new cluster if executing against real ocm
		if ocmConfig.MockMode != ocm.MockModeEmulateServer {
			pErr := PersistClusterStruct(*foundCluster, api.ClusterProvisioning)
			if pErr != nil {
				t.Log(fmt.Sprintf("Unable to persist struct for cluster: %s", foundCluster.ID))
			}
		}
	}

	if expectedStatus != nil {
		_, err := WaitForClusterStatus(h.DBFactory(), &clusterService, foundCluster.ClusterID, *expectedStatus)
		if err != nil {
			return "", ocmErrors.GeneralError("error waiting for cluster '%s' to reach '%s': %v", foundCluster.ClusterID, *expectedStatus, err)
		}
	}

	return foundCluster.ClusterID, nil
}

func findFirstValidCluster(h *test.Helper) (*api.Cluster, *ocmErrors.ServiceError) {
	var clusterService services.ClusterService
	h.Env.MustResolveAll(&clusterService)

	foundClusters, svcErr := clusterService.FindAllClusters(services.FindClusterCriteria{
		Region:   mocks.MockCluster.Region().ID(),
		Provider: mocks.MockCluster.CloudProvider().ID(),
		MultiAZ:  mocks.MockMultiAZ,
	})
	if svcErr != nil {
		return nil, svcErr
	}
	var foundCluster *api.Cluster
	for _, c := range foundClusters {
		if c.Status.String() != api.ClusterDeprovisioning.String() {
			foundCluster = c
			return foundCluster, nil
		}
	}
	return foundCluster, nil
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string, t *testing.T) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		t.Log(fmt.Sprintf("%s not found", filename))
		return false
	}
	return !info.IsDir()
}

// PersistClusterStruct to json file to be reused by tests that require a running cluster
func PersistClusterStruct(cluster api.Cluster, status api.ClusterStatus) error {
	// We intentionally always persist the cluster status as ClusterProvisioned if its ready (ClusterProvisioning otherwise)
	// with the aim of letting cluster reconciler in other tests reach
	// the real status depending on the configured environment settings
	cluster.Status = status
	file, err := json.MarshalIndent(cluster, "", " ")
	if err != nil {
		return ocmErrors.GeneralError(fmt.Sprintf("Failed to marshal cluster struct details to a file: %v", err))
	}
	err = ioutil.WriteFile(testClusterPath, file, 0644)
	if err != nil {
		return ocmErrors.GeneralError(fmt.Sprintf("Failed to persist cluster struct details to a file: %v", err))
	}
	return nil
}

// get clusterID from file if exists. Any issues with reading content of the file results in returning
// an empty string with an error. If no file found - return empty string and nil as an error
func readClusterDetailsFromFile(h *test.Helper, t *testing.T) (string, error) {
	if fileExists(testClusterPath, t) {
		file, err := ioutil.ReadFile(testClusterPath)
		if err != nil {
			return "", ocmErrors.GeneralError(fmt.Sprintf("Failed to ReadFile with cluster details: %v", err))
		}

		cluster := &api.Cluster{}

		marshalErr := json.Unmarshal([]byte(file), cluster)
		if marshalErr != nil {
			return "", ocmErrors.GeneralError(fmt.Sprintf("Failed to Unmarshal cluster details from file: %v", marshalErr))
		}

		// ignore certain fields, these should be set by each test if needed
		cluster.AvailableDinosaurOperatorVersions = nil

		dbConn := h.DBFactory().New()
		if err := dbConn.FirstOrCreate(cluster, &api.Cluster{ClusterID: cluster.ClusterID}).Error; err != nil {
			return "", ocmErrors.GeneralError(fmt.Sprintf("Failed to save cluster details to database: %v", err))
		}
		return cluster.ClusterID, nil
	}
	return "", nil
}

func RemoveClusterFile(t *testing.T) {
	if fileExists(testClusterPath, t) {
		if err := os.Remove(testClusterPath); err != nil {
			t.Errorf("failed to delete file %v due to error %v", testClusterPath, err)
		}
	} else {
		t.Logf("cluster file %s does not exist", testClusterPath)
	}
}

func getMetrics(t *testing.T) string {
	metricsConfig := server.NewMetricsConfig()
	metricsAddress := metricsConfig.BindAddress
	var metricsURL string
	if metricsConfig.EnableHTTPS {
		metricsURL = fmt.Sprintf("https://%s/metrics", metricsAddress)
	} else {
		metricsURL = fmt.Sprintf("http://%s/metrics", metricsAddress)
	}
	response, err := http.Get(metricsURL)
	if err != nil {
		t.Fatalf("failed to get response from %s: %s", metricsURL, err.Error())
	}
	defer response.Body.Close()

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("failed to read response data: %s", err.Error())
	}

	responseString := string(responseData)
	return responseString
}

// CheckMetricExposed - checks whether metric is exposed in the metrics URL
func CheckMetricExposed(h *test.Helper, t *testing.T, metric string) {
	resp := getMetrics(t)
	Expect(strings.Contains(resp, metric)).To(Equal(true))
}

// CheckMetric - check if the given metric exists in the metrics data
func CheckMetric(h *test.Helper, t *testing.T, metric string, exist bool) {
	resp := getMetrics(t)
	Expect(strings.Contains(resp, metric)).To(Equal(exist))
}
