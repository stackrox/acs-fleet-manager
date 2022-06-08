package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"k8s.io/client-go/rest"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

var cfg *rest.Config
var k8sClient client.Client

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
