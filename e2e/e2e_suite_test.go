package e2e

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	cfg             *rest.Config
	k8sClient       client.Client
	routeService    *k8s.RouteService
	dnsEnabled      bool
	routesEnabled   bool
	route53Client   *route53.Route53
	waitTimeout     = getWaitTimeout()
	dpCloudProvider = getEnvDefault("DP_CLOUD_PROVIDER", "standalone")
	dpRegion        = getEnvDefault("DP_REGION", "standalone")
)

const defaultTimeout = 5 * time.Minute

func getWaitTimeout() time.Duration {
	timeoutStr, ok := os.LookupEnv("WAIT_TIMEOUT")
	if ok {
		timeout, err := time.ParseDuration(timeoutStr)
		if err == nil {
			return timeout
		}
		fmt.Printf("Error parsing timeout, using default timeout %v: %s\n", defaultTimeout, err)
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
	routeService = k8s.NewRouteService(k8sClient)
	var err error
	routesEnabled, err = k8s.IsRoutesResourceEnabled()
	Expect(err).ToNot(HaveOccurred())

	var accessKey, secretKey string
	dnsEnabled, accessKey, secretKey = isDNSEnabled(routesEnabled)

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

var _ = AfterSuite(func() {
})

func isDNSEnabled(routesEnabled bool) (bool, string, string) {
	accessKey := os.Getenv("ROUTE53_ACCESS_KEY")
	secretKey := os.Getenv("ROUTE53_SECRET_ACCESS_KEY")
	enableExternal := os.Getenv("ENABLE_DINOSAUR_EXTERNAL_CERTIFICATE")
	dnsEnabled := accessKey != "" &&
		secretKey != "" &&
		enableExternal != "" && routesEnabled
	return dnsEnabled, accessKey, secretKey
}
