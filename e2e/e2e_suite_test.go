package e2e

import (
	"context"
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
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	cfg                   *rest.Config
	k8sClient             ctrlClient.Client
	routeService          *k8s.RouteService
	dnsEnabled            bool
	routesEnabled         bool
	route53Client         *route53.Route53
	waitTimeout           = getWaitTimeout()
	dpCloudProvider       = getEnvDefault("DP_CLOUD_PROVIDER", "standalone")
	dpRegion              = getEnvDefault("DP_REGION", "standalone")
	authType              = "OCM"
	fleetManagerEndpoint  = "http://localhost:8000"
	runAuthTests          bool
	runCentralTests       bool
	runCanaryUpgradeTests bool
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
		t.Skip("Skip e2e tests. Set RUN_E2E=true env variable to enable e2e tests.")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "RHACS ManagedServices Suite")
}

// TODO: Deploy fleet-manager, fleetshard-sync and database into a cluster
var _ = BeforeSuite(func() {
	k8sClient = k8s.CreateClientOrDie()
	routeService = k8s.NewRouteService(k8sClient)
	var err error
	routesEnabled, err = k8s.IsRoutesResourceEnabled(k8sClient)
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

	if val := os.Getenv("AUTH_TYPE"); val != "" {
		authType = val
	}
	GinkgoWriter.Printf("AUTH_TYPE: %q\n", authType)

	if val := os.Getenv("FLEET_MANAGER_ENDPOINT"); val != "" {
		fleetManagerEndpoint = val
	}
	GinkgoWriter.Printf("FLEET_MANAGER_ENDPOINT: %q\n", fleetManagerEndpoint)

	runAuthTests = enableTestsGroup("Auth", "RUN_AUTH_E2E", "false")
	runCentralTests = enableTestsGroup("Central", "RUN_CENTRAL_E2E", "true")
	runCanaryUpgradeTests = enableTestsGroup("CanaryUpgrade", "RUN_CANARY_UPGRADE_E2E", "true")
})

var _ = AfterSuite(func() {
})

func enableTestsGroup(testName string, envName string, defaultValue string) bool {
	if val := getEnvDefault(envName, defaultValue); val == "true" {
		GinkgoWriter.Printf("Executing %s tests", testName)
		return true
	} else {
		GinkgoWriter.Printf("Skipping %s tests. Set %s=true to run these tests", testName, envName)
	}
	return false
}

func isDNSEnabled(routesEnabled bool) (bool, string, string) {
	accessKey := os.Getenv("ROUTE53_ACCESS_KEY")
	secretKey := os.Getenv("ROUTE53_SECRET_ACCESS_KEY")
	enableExternal := os.Getenv("ENABLE_CENTRAL_EXTERNAL_CERTIFICATE")
	dnsEnabled := accessKey != "" &&
		secretKey != "" &&
		enableExternal != "" && routesEnabled
	return dnsEnabled, accessKey, secretKey
}

func assertCentralRequestStatus(ctx context.Context, client *fleetmanager.Client, id string, status string) func() error {
	return func() error {
		centralRequest, _, err := client.PublicAPI().GetCentralById(ctx, id)
		if err != nil {
			return err
		}
		if centralRequest.Status != status {
			return fmt.Errorf("expected centralRequest status %s, got %s", status, centralRequest.Status)
		}
		return nil
	}
}

func obtainCentralRequest(ctx context.Context, client *fleetmanager.Client, id string, request *public.CentralRequest) func() error {
	return func() error {
		centralRequest, _, err := client.PublicAPI().GetCentralById(ctx, id)
		if err != nil {
			return err
		}
		*request = centralRequest
		return nil
	}
}

func assertStoredSecrets(ctx context.Context, client *fleetmanager.Client, centralRequestID string, expected []string) func() error {
	return func() error {
		privateCentral, _, err := client.PrivateAPI().GetCentral(ctx, centralRequestID)
		if err != nil {
			return err
		}
		if len(privateCentral.Metadata.SecretsStored) != len(expected) {
			return fmt.Errorf("unexpected number of secrets, want: %d, got: %d", len(expected), len(privateCentral.Metadata.SecretsStored))
		}
		Expect(privateCentral.Metadata.SecretsStored).Should(ContainElements(expected)) // pragma: allowlist secret
		return nil
	}
}

func assertCentralCRExists(ctx context.Context, central *v1alpha1.Central, namespace, name string) func() error {
	return assertObjectExists(ctx, central, namespace, name)
}

func assertSecretExists(ctx context.Context, secret *v1.Secret, namespace, name string) func() error {
	return assertObjectExists(ctx, secret, namespace, name)
}

func assertNamespaceExists(ctx context.Context, ns *v1.Namespace, name string) func() error {
	return assertObjectExists(ctx, ns, "", name)
}

func assertObjectExists(ctx context.Context, obj ctrlClient.Object, namespace, name string) func() error {
	return func() error {
		key := ctrlClient.ObjectKey{Name: name, Namespace: namespace}
		err := k8sClient.Get(ctx, key, obj)
		if err != nil {
			return err
		}
		return nil
	}
}

func deleteCentralByID(ctx context.Context, client *fleetmanager.Client, id string) func() error {
	return func() error {
		_, err := client.PublicAPI().DeleteCentralById(ctx, id, true)
		return err
	}
}

func assertCentralCRDeleted(ctx context.Context, namespace, name string) func() error {
	central := &v1alpha1.Central{}
	return assertObjectDeleted(ctx, central, namespace, name)
}

func assertDeploymentDeleted(ctx context.Context, namespace, name string) func() error {
	deployment := &appsv1.Deployment{}
	return assertObjectDeleted(ctx, deployment, namespace, name)
}

func assertNamespaceDeleted(ctx context.Context, name string) func() error {
	ns := &v1.Namespace{}
	return assertObjectDeleted(ctx, ns, "", name)
}

func assertObjectDeleted(ctx context.Context, obj ctrlClient.Object, namespace, name string) func() error {
	return func() error {
		key := ctrlClient.ObjectKey{Name: name, Namespace: namespace}
		err := k8sClient.Get(ctx, key, obj)
		if err != nil {
			if apiErrors.IsNotFound(err) {
				return nil
			}
			return err
		}
		return fmt.Errorf("%s %s/%s still exists", obj.GetObjectKind().GroupVersionKind().Kind, namespace, name)
	}
}

func assertDeploymentHealthyReplicas(ctx context.Context, namespace, name string, replicas int32) func() error {
	return func() error {
		deployment := &appsv1.Deployment{}
		key := ctrlClient.ObjectKey{Name: name, Namespace: namespace}
		err := k8sClient.Get(ctx, key, deployment)
		if err != nil {
			return err
		}
		if *deployment.Spec.Replicas != replicas {
			return fmt.Errorf("expected deployment %s/%s replicas %d, got %d. ready=%d. unavailable=%d", namespace, name, replicas, *deployment.Spec.Replicas, deployment.Status.ReadyReplicas, deployment.Status.UnavailableReplicas)
		}
		if deployment.Status.ReadyReplicas != replicas {
			return fmt.Errorf("expected deployment %s/%s ready replicas %d, got %d. ready=%d. unavailable=%d", namespace, name, replicas, deployment.Status.ReadyReplicas, deployment.Status.ReadyReplicas, deployment.Status.UnavailableReplicas)
		}
		return nil
	}
}

func assertReencryptIngressRouteExist(ctx context.Context, namespace string, route *openshiftRouteV1.RouteIngress) func() error {
	return func() error {
		reencryptIngress, err := routeService.FindReencryptIngress(ctx, namespace)
		if err != nil {
			return fmt.Errorf("failed finding reencrypt ingress in namespace %s: %v", namespace, err)
		}
		if reencryptIngress == nil {
			return fmt.Errorf("reencrypt ingress in namespace %s not found", namespace)
		}
		*route = *reencryptIngress
		return nil
	}
}

func assertReencryptRouteExist(ctx context.Context, namespace string, route *openshiftRouteV1.Route) func() error {
	return func() error {
		reencryptRoute, err := routeService.FindReencryptRoute(ctx, namespace)
		if err != nil {
			return fmt.Errorf("failed finding reencrypt route: %v", err)
		}
		if reencryptRoute == nil {
			return fmt.Errorf("reencrypt route in namespace %s not found", namespace)
		}
		*route = *reencryptRoute
		return nil
	}
}

func assertPassthroughRouteExist(ctx context.Context, namespace string, route *openshiftRouteV1.Route) func() error {
	return func() error {
		passthroughRoute, err := routeService.FindPassthroughRoute(ctx, namespace)
		if err != nil {
			return fmt.Errorf("failed finding passthrough route in namespace %s: %v", namespace, err)
		}
		if passthroughRoute == nil {
			return fmt.Errorf("passthrough route not found in namespace %s", namespace)
		}
		*route = *passthroughRoute
		return nil
	}
}

func SkipIf(condition bool, message string) {
	if condition {
		Skip(message, 1)
	}
}
