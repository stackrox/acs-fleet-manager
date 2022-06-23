package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"k8s.io/client-go/rest"
	"fmt"
	"os"
	"time"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

var cfg *rest.Config
var k8sClient client.Client

const defaultTimeout = 5 * time.Minute
var waitTimeout = getWaitTimeout()
var dpCloudProvider = getEnvDefault("DP_CLOUD_PROVIDER", "standalone")
var dpRegion = getEnvDefault("DP_REGION", "standalone")

func getWaitTimeout() time.Duration {
	timeoutStr, ok := os.LookupEnv("WAIT_TIMEOUT")
	if ok {
		timeout, err := time.ParseDuration(timeoutStr)
		if err == nil {
			return timeout
		} else {
			fmt.Printf("Error parsing timeout, using default timeout %v: %s\n", defaultTimeout, err)
		}
	}
	return defaultTimeout
}

func getEnvDefault(key, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}
	return value
}

func TestE2E(t *testing.T) {
	if os.Getenv("RUN_E2E") != "true" {
		t.Skip("Skip e2e tests. Set RUN_E2E=1 env variable to enable e2e tests.")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "RHACS ManagedServices Suite")
}

//TODO: Deploy fleet-manager, fleetshard-sync and database into a cluster
var _ = BeforeSuite(func() {
	k8sClient = k8s.CreateClientOrDie()
})

var _ = AfterSuite(func() {
})
