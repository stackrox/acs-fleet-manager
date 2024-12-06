package multicluster

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"k8s.io/client-go/tools/clientcmd"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	k8sClientCluster1    ctrlClient.Client
	k8sClientCluster2    ctrlClient.Client
	fleetManagerEndpoint = "http://localhost:8000"
)

func TestMulticlusterE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ACSCS Multicluster Suite")
}

var _ = BeforeSuite(func() {
	cluster1ConfigFile := os.Getenv("CLUSTER_1_KUBECONFIG")
	cluster2ConfigFile := os.Getenv("CLUSTER_2_KUBECONFIG")
	Expect(cluster1ConfigFile).ToNot(BeEmpty())
	Expect(cluster2ConfigFile).ToNot(BeEmpty())

	configC1, err := os.ReadFile(cluster1ConfigFile)
	Expect(err).ToNot(HaveOccurred())
	restConfigC1, err := clientcmd.RESTConfigFromKubeConfig(configC1)
	Expect(err).ToNot(HaveOccurred())
	k8sClientCluster1 = k8s.CreateClientWithConfigOrDie(restConfigC1)

	configC2, err := os.ReadFile(cluster2ConfigFile)
	Expect(err).ToNot(HaveOccurred())
	restConfigC2, err := clientcmd.RESTConfigFromKubeConfig(configC2)
	Expect(err).ToNot(HaveOccurred())
	k8sClientCluster2 = k8s.CreateClientWithConfigOrDie(restConfigC2)

	fmOverride := os.Getenv("FM_URL")
	if fmOverride != "" {
		fleetManagerEndpoint = fmOverride
	}

})
