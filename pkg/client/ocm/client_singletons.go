package ocm

import (
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"sync"
)

var (
	onceAMSClient sync.Once
	ocmAMSClient  AMSClient

	onceClusterManagmentClient sync.Once
	ocmClusterManagmentClient  ClusterManagementClient

	ocmConfig = GetOCMConfig()
)

// SingletonAMSClient returns AMSClient
func SingletonAMSClient() AMSClient {
	onceAMSClient.Do(func() {
		if ocmConfig.EnableMock {
			ocmAMSClient = NewMockClient()
		}

		conn, _, err := NewOCMConnection(ocmConfig, ocmConfig.AmsURL)
		if err != nil {
			logger.Logger.Error(err)
		}
		ocmAMSClient = NewClient(conn)
	})
	return ocmAMSClient
}

// SingletonClusterManagementClient returns ClusterManagementClient
func SingletonClusterManagementClient() ClusterManagementClient {
	onceClusterManagmentClient.Do(func() {
		if ocmConfig.EnableMock {
			ocmClusterManagmentClient = NewMockClient()
		}

		conn, _, err := NewOCMConnection(ocmConfig, ocmConfig.BaseURL)
		if err != nil {
			logger.Logger.Error(err)
		}
		ocmClusterManagmentClient = NewClient(conn)
	})

	return ocmClusterManagmentClient
}
