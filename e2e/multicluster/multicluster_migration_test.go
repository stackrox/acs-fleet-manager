package multicluster

import (
	"context"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/stackrox/acs-fleet-manager/e2e/dns"
	"github.com/stackrox/acs-fleet-manager/e2e/testutil"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	fmImpl "github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager/impl"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	dpCloudProvider = "standalone"
	dpRegion        = "standalone"
	// the cluster ids configured for given clusters in fleet-manager's cluster config
	cluster1ID = "cluster-1-id"
	cluster2ID = "cluster-2-id"
)

var (
	waitTimeout         = testutil.GetWaitTimeout()
	extendedWaitTimeout = testutil.GetWaitTimeout() * 3
	defaultPolling      = 5 * time.Second
)

var _ = Describe("Central Migration Test", Ordered, func() {
	var fleetmanagerClient *fleetmanager.Client
	var fleetmanagerAdminClient fleetmanager.AdminAPI
	var centralRequest public.CentralRequest
	var namespaceName string
	var notes []string
	BeforeAll(func() {
		options := fmImpl.OptionFromEnv()
		auth, err := fmImpl.NewStaticAuth(context.Background(), fmImpl.StaticOption{StaticToken: options.Static.StaticToken})
		Expect(err).ToNot(HaveOccurred())
		fleetmanagerClient, err = fmImpl.NewClient(fleetManagerEndpoint, auth)
		Expect(err).ToNot(HaveOccurred())

		adminStaticToken := os.Getenv("STATIC_TOKEN_ADMIN")
		adminAuth, err := fmImpl.NewStaticAuth(context.Background(), fmImpl.StaticOption{StaticToken: adminStaticToken})
		Expect(err).ToNot(HaveOccurred())
		adminClient, err := fmImpl.NewClient(fleetManagerEndpoint, adminAuth)
		Expect(err).ToNot(HaveOccurred())
		fleetmanagerAdminClient = adminClient.AdminAPI()
	})

	AfterAll(func() {
		// if the Id is empty we've never successfully created a CentralRequest, thus no cleanup necessary
		if dnsEnabled && centralRequest.Id != "" {
			dns.CleanupCentralRequestRecords(route53Client, centralRequest)
		}

		for _, note := range notes {
			GinkgoWriter.Println(note)
		}
	})

	It("should create a CentralRequest", func() {
		resp, _, err := fleetmanagerClient.PublicAPI().CreateCentral(context.Background(), true, public.CentralRequestPayload{
			CloudProvider: dpCloudProvider,
			Region:        dpRegion,
			MultiAz:       true,
			Name:          "migration-test",
		})
		Expect(err).NotTo(HaveOccurred())
		centralRequest = resp
		namespaceName, err = services.FormatNamespace(centralRequest.Id)
		Expect(err).NotTo(HaveOccurred())

		notes = []string{
			fmt.Sprintf("Central name: %s", centralRequest.Name),
			fmt.Sprintf("Central ID: %s", centralRequest.Id),
		}
	})

	Describe("CentralRequest pre migration", Ordered, func() {
		It("should be assigned to cluster1", func() {
			assertClusterAssignment(cluster1ID, centralRequest.Id, fleetmanagerAdminClient)
		})

		It("should reach the ready state", func() {
			Eventually(testutil.AssertCentralRequestReady(context.Background(), fleetmanagerClient, centralRequest.Id)).
				WithTimeout(extendedWaitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should have DNS CNAME records for cluster1 routes", func() {
			testutil.SkipIf(!dnsEnabled, testutil.SkipDNSMsg)
			dnsRecordsLoader := dns.NewRecordsLoader(route53Client, centralRequest)
			routeService := k8s.NewRouteService(cluster1KubeClient, routeConfig)

			var reencryptIngress openshiftRouteV1.RouteIngress
			Eventually(testutil.AssertReencryptIngressRouteExist(context.Background(), routeService, namespaceName, &reencryptIngress)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())

			Eventually(dnsRecordsLoader.LoadDNSRecords).
				WithTimeout(waitTimeout).
				WithPolling(3 * defaultPolling).
				Should(HaveLen(len(dnsRecordsLoader.CentralDomainNames)))

			testutil.AssertDNSMatchesRouter(dnsRecordsLoader.CentralDomainNames, dnsRecordsLoader.LastResult, &reencryptIngress)
		})
	})

	Describe("Tenant namespace pre migration", func() {
		It("should exist on cluster1", func() {
			ns, err := getNamespace(namespaceName, cluster1KubeClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(ns).NotTo(BeNil())
		})
		It("should not exist on cluster2", func() {
			_, err := getNamespace(namespaceName, cluster2KubeClient)
			Expect(apiErrors.IsNotFound(err)).To(BeTrue())
		})
	})

	It("should trigger CentralRequest migration", func() {
		_, err := fleetmanagerAdminClient.AssignCentralCluster(context.Background(), centralRequest.Id, private.CentralAssignClusterRequest{
			ClusterId: cluster2ID,
		})
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("CentralRequest post migration", Ordered, func() {
		It("should be assigned to cluster2", func() {
			assertClusterAssignment(cluster1ID, centralRequest.Id, fleetmanagerAdminClient)
		})
		It("should reach the ready state", func() {
			Eventually(testutil.AssertCentralRequestReady(context.Background(), fleetmanagerClient, centralRequest.Id)).
				WithTimeout(extendedWaitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})
		It("should have DNS CNAME records for cluster2 routes", func() {
			testutil.SkipIf(!dnsEnabled, testutil.SkipDNSMsg)
			dnsRecordsLoader := dns.NewRecordsLoader(route53Client, centralRequest)
			routeService := k8s.NewRouteService(cluster2KubeClient, routeConfig)

			var reencryptIngress openshiftRouteV1.RouteIngress
			Eventually(testutil.AssertReencryptIngressRouteExist(context.Background(), routeService, namespaceName, &reencryptIngress)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())

			Eventually(dnsRecordsLoader.LoadDNSRecords).
				WithTimeout(waitTimeout).
				WithPolling(3 * defaultPolling).
				Should(HaveLen(len(dnsRecordsLoader.CentralDomainNames)))

			testutil.AssertDNSMatchesRouter(dnsRecordsLoader.CentralDomainNames, dnsRecordsLoader.LastResult, &reencryptIngress)
		})
	})

	Describe("Tenant namespace post migration", func() {
		It("should not exist on cluster1", func() {
			// Using Eventually here because fleetshard-sync on cluster1 can take a while to cleanup the NS
			Eventually(func() error {
				_, err := getNamespace(namespaceName, cluster1KubeClient)
				if apiErrors.IsNotFound(err) {
					return nil
				}

				if err == nil {
					return fmt.Errorf("namespace: %q still exists", namespaceName)
				}

				return err
			}).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should exist on cluster2", func() {
			ns, err := getNamespace(namespaceName, cluster2KubeClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(ns).NotTo(BeNil())
		})
	})

	It("should delete CentralRequest", func() {
		_, err := fleetmanagerClient.PublicAPI().DeleteCentralById(context.Background(), centralRequest.Id, true)
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() (int, error) {
			_, res, err := fleetmanagerClient.PublicAPI().GetCentralById(context.Background(), centralRequest.Id)
			if res != nil {
				return res.StatusCode, err
			}

			return -1, err
		}).
			WithTimeout(waitTimeout).
			WithPolling(defaultPolling).
			Should(Equal(404))
	})

})

func assertClusterAssignment(expectedClusterID string, centralID string, adminAPI fleetmanager.AdminAPI) {
	var clusterAssignment string
	Eventually(func() (err error) {
		// assert the cluster ID outside the Eventually, since once we have a non-empty
		// clusterAssignment it will not change so there is no need to keep polling
		clusterAssignment, err = getClusterAssignment(centralID, adminAPI)
		return err
	}).
		WithTimeout(waitTimeout).
		WithPolling(defaultPolling).
		Should(Succeed())

	Expect(clusterAssignment).To(Equal(expectedClusterID))
}

func getClusterAssignment(centralID string, adminAPI fleetmanager.AdminAPI) (string, error) {
	centralList, _, err := adminAPI.GetCentrals(context.Background(), nil)
	if err != nil {
		return "", err
	}

	if len(centralList.Items) == 0 {
		return "", fmt.Errorf("central list received by admin API is empty")
	}

	var clusterID string
	var tenantExists bool
	for _, central := range centralList.Items {
		if central.Id == centralID {
			tenantExists = true
			clusterID = central.ClusterId
		}
	}

	if !tenantExists {
		return "", fmt.Errorf("CentralRequest with id: %q not found in admin API list", centralID)
	}

	if clusterID == "" {
		return "", fmt.Errorf("CentralRequest returned with no assigned clusterID")
	}

	return clusterID, nil
}

func getNamespace(name string, kubeClient ctrlClient.Client) (*corev1.Namespace, error) {
	var namespace *corev1.Namespace
	if err := kubeClient.Get(context.Background(), ctrlClient.ObjectKey{Name: name}, namespace); err != nil {
		return nil, fmt.Errorf("getting namespace %q: %w", name, err)
	}

	return namespace, nil
}
