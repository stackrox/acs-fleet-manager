package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/stackrox/acs-fleet-manager/e2e/testutil"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/rox/operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	k8sClient             ctrlClient.Client
	routeService          *k8s.RouteService
	dnsEnabled            bool
	routesEnabled         bool
	waitTimeout           = testutil.GetWaitTimeout()
	extendedWaitTimeout   = testutil.GetWaitTimeout() * 3
	dpCloudProvider       = getEnvDefault("DP_CLOUD_PROVIDER", "standalone")
	dpRegion              = getEnvDefault("DP_REGION", "standalone")
	fleetManagerEndpoint  = "http://localhost:8000"
	runAuthTests          bool
	runCentralTests       bool
	runCanaryUpgradeTests bool
)

const defaultTimeout = 5 * time.Minute

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

func assertStoredSecrets(ctx context.Context, privateAPI fleetmanager.PrivateAPI, centralRequestID string, expected []string) func() error {
	return func() error {
		privateCentral, _, err := privateAPI.GetCentral(ctx, centralRequestID)
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

func deleteCentralByID(ctx context.Context, client *fleetmanager.Client, id string) error {
	_, err := client.PublicAPI().DeleteCentralById(ctx, id, true)
	return err
}

func assertCentralCRDeleted(ctx context.Context, namespace, name string) func() error {
	central := &v1alpha1.Central{}
	return assertObjectDeleted(ctx, central, namespace, name)
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
		return fmt.Errorf("%T %s/%s still exists", obj, namespace, name)
	}
}

func assertRouteExists(ctx context.Context, namespace string, expectedTermination openshiftRouteV1.TLSTerminationType, expectedHost string) func(g Gomega) {
	return func(g Gomega) {
		routes := &openshiftRouteV1.RouteList{}
		err := k8sClient.List(ctx, routes, ctrlClient.InNamespace(namespace))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(routes.Items).To(ContainElement(SatisfyAll(
			WithTransform(getRouteTermination, Equal(expectedTermination)),
			WithTransform(getRouteHost, Equal(expectedHost)),
		)))
	}
}

func getRouteTermination(route openshiftRouteV1.Route) openshiftRouteV1.TLSTerminationType {
	return route.Spec.TLS.Termination
}

func getRouteHost(route openshiftRouteV1.Route) string {
	return route.Spec.Host
}
