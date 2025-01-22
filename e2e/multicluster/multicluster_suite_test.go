package multicluster

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/e2e/testutil"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"k8s.io/client-go/tools/clientcmd"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	cluster1KubeClient ctrlClient.Client
	cluster2KubeClient ctrlClient.Client

	fleetManagerEndpoint = "http://localhost:8000"
	route53Client        *route53.Route53
	dnsEnabled           bool

	routeConfig = &config.RouteConfig{
		ConcurrentTCP: 32,
		RateHTTP:      128,
		RateTCP:       16,
	}
)

func TestMulticlusterE2E(t *testing.T) {
	if os.Getenv("RUN_MULTICLUSTER_E2E") != "true" {
		t.Skip("Skip multicluster e2e tests. Set RUN_MULTICLUSTER_E2E=true env variable to enable e2e tests.")
	}

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
	cluster1KubeClient = k8s.CreateClientWithConfigOrDie(restConfigC1)

	configC2, err := os.ReadFile(cluster2ConfigFile)
	Expect(err).ToNot(HaveOccurred())
	restConfigC2, err := clientcmd.RESTConfigFromKubeConfig(configC2)
	Expect(err).ToNot(HaveOccurred())
	cluster2KubeClient = k8s.CreateClientWithConfigOrDie(restConfigC2)

	fmOverride := os.Getenv("FM_URL")
	if fmOverride != "" {
		fleetManagerEndpoint = fmOverride
	}

	routesEnabled, err := k8s.IsRoutesResourceEnabled(cluster1KubeClient)
	Expect(err).ToNot(HaveOccurred())

	var accessKey, secretKey string
	dnsEnabled, accessKey, secretKey = testutil.DNSConfiguration(routesEnabled)

	if dnsEnabled {
		creds := credentials.NewStaticCredentials(
			accessKey,
			secretKey,
			"")
		sess, err := session.NewSession(aws.NewConfig().WithCredentials(creds))
		Expect(err).ToNot(HaveOccurred())

		route53Client = route53.New(sess)
	}
})
