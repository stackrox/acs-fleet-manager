package e2e

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/stackrox/acs-fleet-manager/e2e/dns"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newCentralName() string {
	rnd := make([]byte, 8)
	_, err := rand.Read(rnd)

	if err != nil {
		panic(fmt.Errorf("reading random bytes for unique central name: %w", err))
	}
	rndString := hex.EncodeToString(rnd)

	return fmt.Sprintf("%s-%s", "e2e", rndString)
}

const (
	defaultPolling = 1 * time.Second
	skipRouteMsg   = "route resource is not known to test cluster"
	skipDNSMsg     = "external DNS is not enabled for this test run"
)

var _ = Describe("Central", Ordered, func() {
	var client *fleetmanager.Client
	var adminAPI fleetmanager.AdminAPI
	var notes []string
	var ctx = context.Background()

	BeforeEach(func() {
		Expect(restoreDefaultGitopsConfig).To(Succeed())
	})

	BeforeEach(func() {
		SkipIf(!runCentralTests, "Skipping Central tests")

		option := fleetmanager.OptionFromEnv()
		auth, err := fleetmanager.NewStaticAuth(context.Background(), fleetmanager.StaticOption{StaticToken: option.Static.StaticToken})
		Expect(err).ToNot(HaveOccurred())
		client, err = fleetmanager.NewClient(fleetManagerEndpoint, auth)
		Expect(err).ToNot(HaveOccurred())

		adminStaticToken := os.Getenv("STATIC_TOKEN_ADMIN")
		adminAuth, err := fleetmanager.NewStaticAuth(context.Background(), fleetmanager.StaticOption{StaticToken: adminStaticToken})
		Expect(err).ToNot(HaveOccurred())
		adminClient, err := fleetmanager.NewClient(fleetManagerEndpoint, adminAuth)
		Expect(err).ToNot(HaveOccurred())
		adminAPI = adminClient.AdminAPI()

		GinkgoWriter.Printf("Current time: %s\n", time.Now().String())
		printNotes(notes)
	})

	Describe("should be created and deployed to k8s", Ordered, func() {

		var centralRequestID string
		var centralRequestName string
		var namespaceName string

		BeforeAll(func() {
			resp, _, err := client.PublicAPI().CreateCentral(ctx, true, public.CentralRequestPayload{
				CloudProvider: dpCloudProvider,
				MultiAz:       true,
				Name:          newCentralName(),
				Region:        dpRegion,
			})
			Expect(err).To(BeNil())
			centralRequestID = resp.Id
			centralRequestName = resp.Name
			notes = []string{
				fmt.Sprintf("Central name: %s", resp.Name),
				fmt.Sprintf("Central ID: %s", resp.Id),
			}
			printNotes(notes)
			namespaceName, err = services.FormatNamespace(centralRequestID)
			Expect(err).To(BeNil())
			Expect(resp.Status).To(Equal(constants.CentralRequestStatusAccepted.String()))
		})

		It("should transition central request state to provisioning", func() {
			provisioning := constants.CentralRequestStatusProvisioning.String()
			Eventually(assertCentralRequestStatus(ctx, client, centralRequestID, provisioning)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should create central namespace", func() {
			var ns corev1.Namespace
			Eventually(assertNamespaceExists(ctx, &ns, namespaceName)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
			_, tenantLabelFound := ns.Labels["rhacs.redhat.com/tenant"]
			Expect(tenantLabelFound).To(BeTrue())
		})

		It("should generate a central-encryption-key secret", func() {
			Eventually(assertSecretExists(ctx, &corev1.Secret{}, namespaceName, "central-encryption-key")).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should create central CR in its namespace on a managed cluster", func() {
			Eventually(assertCentralCRExists(ctx, &v1alpha1.Central{}, namespaceName, centralRequestName)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		// TODO: possible flake. Maybe this test will be executed after the routes are created
		It("should not expose URLs until the routes are created", func() {
			SkipIf(!routesEnabled, skipRouteMsg)
			var centralRequest public.CentralRequest
			Expect(obtainCentralRequest(ctx, client, centralRequestID, &centralRequest)).
				To(Succeed())
			Expect(centralRequest.CentralUIURL).To(BeEmpty())
			Expect(centralRequest.CentralDataURL).To(BeEmpty())
		})

		It("should transition central request state to ready", func() {
			Expect(obtainCentralRequest(ctx, client, centralRequestID, &public.CentralRequest{})).
				To(Succeed())
			Eventually(assertCentralRequestStatus(ctx, client, centralRequestID, constants.CentralRequestStatusReady.String())).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should have created central routes", func() {
			SkipIf(!routesEnabled, skipRouteMsg)

			var centralRequest public.CentralRequest
			Expect(obtainCentralRequest(ctx, client, centralRequestID, &centralRequest)).
				To(Succeed())

			var reencryptRoute openshiftRouteV1.Route
			Eventually(assertReencryptRouteExist(ctx, namespaceName, &reencryptRoute)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())

			centralUIURL, err := url.Parse(centralRequest.CentralUIURL)
			Expect(err).ToNot(HaveOccurred())
			Expect(centralUIURL.Scheme).To(Equal("https"))
			Expect(reencryptRoute.Spec.Host).To(Equal(centralUIURL.Host))
			Expect(reencryptRoute.Spec.TLS.Termination).To(Equal(openshiftRouteV1.TLSTerminationReencrypt))

			var passthroughRoute openshiftRouteV1.Route
			Eventually(assertPassthroughRouteExist(ctx, namespaceName, &passthroughRoute)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())

			centralDataHost, centralDataPort, err := net.SplitHostPort(centralRequest.CentralDataURL)
			Expect(err).ToNot(HaveOccurred())
			Expect(passthroughRoute.Spec.Host).To(Equal(centralDataHost))
			Expect(centralDataPort).To(Equal("443"))
			Expect(passthroughRoute.Spec.TLS.Termination).To(Equal(openshiftRouteV1.TLSTerminationPassthrough))
		})

		It("should have created AWS Route53 records", func() {
			SkipIf(!dnsEnabled, skipDNSMsg)

			var centralRequest public.CentralRequest
			Expect(obtainCentralRequest(ctx, client, centralRequestID, &centralRequest)).
				To(Succeed())

			var reencryptIngress openshiftRouteV1.RouteIngress
			Eventually(assertReencryptIngressRouteExist(ctx, namespaceName, &reencryptIngress)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())

			dnsRecordsLoader := dns.NewRecordsLoader(route53Client, centralRequest)

			Eventually(dnsRecordsLoader.LoadDNSRecords).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(HaveLen(len(dnsRecordsLoader.CentralDomainNames)), "Started at %s", time.Now())

			recordSets := dnsRecordsLoader.LastResult
			for idx, domain := range dnsRecordsLoader.CentralDomainNames {
				recordSet := recordSets[idx]
				Expect(len(recordSet.ResourceRecords)).To(Equal(1))
				record := recordSet.ResourceRecords[0]
				Expect(*recordSet.Name).To(Equal(domain))
				Expect(*record.Value).To(Equal(reencryptIngress.RouterCanonicalHostname)) // TODO use route specific ingress instead of comparing with reencryptIngress for all cases
			}
		})

		It("should spin up an egress proxy with three healthy replicas", func() {
			Eventually(assertDeploymentHealthyReplicas(ctx, namespaceName, "egress-proxy", 3)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should backup important secrets in FM database", func() {
			// We don't expect central-db-password secret here, because it will not be created in case of
			// running with disabled managed DB
			Eventually(assertStoredSecrets(ctx, client, centralRequestID, []string{"central-tls", "central-encryption-key"})).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should set ForceReconcile through admin API", func() {
			cfg := defaultGitopsConfig()
			cfg.Centrals.Overrides = append(cfg.Centrals.Overrides, overrideCentralWithPatch(centralRequestID, forceReconcilePatch()))
			Expect(putGitopsConfig(ctx, cfg)).To(Succeed())
		})

		// TODO(ROX-11368): Add test to eventually reach ready state
		// TODO(ROX-11368): create test to check that Central and Scanner are healthy
		// TODO(ROX-11368): Create test to check Central is correctly exposed

		It("should restore secrets and deployment on namespace delete", func() {
			if createdCentral == nil {
				Fail("central not created")
			}

			previousNamespace := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
			}

			err := k8sClient.Get(context.Background(), ctrlClient.ObjectKeyFromObject(&previousNamespace), &previousNamespace)
			Expect(err).To(BeNil())

			// Using managedDB false here because e2e don't run with managed postgresql
			secretBackup := k8s.NewSecretBackup(k8sClient, false)
			expectedSecrets, err := secretBackup.CollectSecrets(context.Background(), namespaceName)
			Expect(err).To(BeNil())

			previousCreationTime := previousNamespace.CreationTimestamp
			err = k8sClient.Delete(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespaceName}})
			Expect(err).To(BeNil())

			Eventually(func() error {
				newNamespace := corev1.Namespace{}
				err := k8sClient.Get(context.Background(), ctrlClient.ObjectKey{Name: namespaceName}, &newNamespace)
				if err != nil {
					return err
				}

				if previousCreationTime.Equal(&newNamespace.CreationTimestamp) {
					return fmt.Errorf("namespace found but was not yet deleted")
				}

				return nil
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Succeed())

			actualSecrets := map[string]*corev1.Secret{}
			Eventually(func() error {
				secrets, err := secretBackup.CollectSecrets(context.Background(), namespaceName)
				if err != nil {
					return err
				}
				actualSecrets = secrets // pragma: allowlist secret
				return nil
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Succeed())

			Expect(actualSecrets).ToNot(BeEmpty())
			Expect(len(actualSecrets)).To(Equal(len(expectedSecrets)))
			for secretName := range expectedSecrets { // pragma: allowlist secret
				actualData := actualSecrets[secretName].StringData
				expectedData := expectedSecrets[secretName].StringData
				Expect(actualData).To(Equal(expectedData))
			}
		})

		It("should transition central to deprovisioning state", func() {
			Expect(deleteCentralByID(ctx, client, centralRequestID)).
				To(Succeed())
			deprovisioning := constants.CentralRequestStatusDeprovision.String()
			Eventually(assertCentralRequestStatus(ctx, client, centralRequestID, deprovisioning)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should delete central CR", func() {
			Eventually(assertCentralCRDeleted(ctx, namespaceName, centralRequestName)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should delete the egress proxy", func() {
			Eventually(assertDeploymentDeleted(ctx, namespaceName, "egress-proxy")).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should remove central namespace", func() {
			Eventually(assertNamespaceDeleted(ctx, namespaceName)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should delete external DNS entries", func() {
			SkipIf(!dnsEnabled, skipDNSMsg)
			var centralRequest public.CentralRequest
			Expect(obtainCentralRequest(ctx, client, centralRequestID, &centralRequest)).
				To(Succeed())
			dnsRecordsLoader := dns.NewRecordsLoader(route53Client, centralRequest)
			Eventually(dnsRecordsLoader.LoadDNSRecords).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(BeEmpty(), "Started at %s", time.Now())
		})

		It("should restore the default gitops config", func() {
			Expect(restoreDefaultGitopsConfig).To(Succeed())
		})
	})

	Describe("should be created and deployed to k8s with admin API", Ordered, func() {
		var centralRequestID string
		var centralRequestName string
		var namespaceName string

		BeforeAll(func() {
			centralName := newCentralName()
			request := private.CentralRequestPayload{
				Name:          centralName,
				MultiAz:       true,
				CloudProvider: dpCloudProvider,
				Region:        dpRegion,
			}
			resp, _, err := adminAPI.CreateCentral(context.TODO(), true, request)
			Expect(err).To(BeNil())
			notes = []string{
				fmt.Sprintf("Central name: %s", resp.Name),
				fmt.Sprintf("Central ID: %s", resp.Id),
			}
			centralRequestID = resp.Id
			namespaceName, err = services.FormatNamespace(centralRequestID)
			Expect(err).To(BeNil())
			Expect(constants.CentralRequestStatusAccepted.String()).To(Equal(resp.Status))
		})

		It("should create central in its namespace on a managed cluster", func() {
			Eventually(assertCentralCRExists(ctx, &v1alpha1.Central{}, namespaceName, centralRequestName)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should transition central's state to ready", func() {
			Eventually(assertCentralRequestStatus(ctx, client, centralRequestID, constants.CentralRequestStatusReady.String())).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should transition central to deprovisioning state when deleting", func() {
			_, err := client.PublicAPI().DeleteCentralById(context.TODO(), centralRequestID, true)
			Expect(err).NotTo(HaveOccurred())
			Eventually(assertCentralRequestStatus(ctx, client, centralRequestID, constants.CentralRequestStatusDeprovision.String())).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should delete central CR", func() {
			Eventually(assertCentralCRDeleted(ctx, namespaceName, centralRequestName)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should remove central namespace", func() {
			Eventually(assertNamespaceDeleted(ctx, namespaceName)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should delete external DNS entries", func() {
			SkipIf(!dnsEnabled, skipDNSMsg)
			var centralRequest public.CentralRequest
			Expect(obtainCentralRequest(ctx, client, centralRequestID, &centralRequest)).
				To(Succeed())
			dnsRecordsLoader := dns.NewRecordsLoader(route53Client, centralRequest)
			Eventually(dnsRecordsLoader.LoadDNSRecords).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(BeEmpty(), "Started at %s", time.Now())
		})

	})

	Describe("should be deployed and can be force-deleted", Ordered, func() {
		var centralRequestID string
		var centralRequestName string
		var namespaceName string

		BeforeAll(func() {
			centralName := newCentralName()
			request := public.CentralRequestPayload{
				Name:          centralName,
				MultiAz:       true,
				CloudProvider: dpCloudProvider,
				Region:        dpRegion,
			}

			resp, _, err := client.PublicAPI().CreateCentral(ctx, true, request)
			Expect(err).To(BeNil())
			centralRequestID = resp.Id
			centralRequestName = resp.Name
			notes = []string{
				fmt.Sprintf("Central name: %s", centralRequestName),
				fmt.Sprintf("Central ID: %s", centralRequestID),
			}
			namespaceName, err = services.FormatNamespace(centralRequestID)
			Expect(err).To(BeNil())
			Expect(resp.Status).To(Equal(constants.CentralRequestStatusAccepted.String()))
		})

		It("should transition central's state to ready", func() {
			Eventually(assertCentralRequestStatus(ctx, client, centralRequestID, constants.CentralRequestStatusReady.String())).
				WithTimeout(extendedWaitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should be deletable in the control-plane database", func() {
			_, err := adminAPI.DeleteDbCentralById(ctx, centralRequestID)
			Expect(err).ToNot(HaveOccurred())
			_, err = adminAPI.DeleteDbCentralById(ctx, centralRequestID)
			Expect(err).To(HaveOccurred())
			central, _, err := client.PublicAPI().GetCentralById(ctx, centralRequestID)
			Expect(err).To(HaveOccurred())
			Expect(central.Id).To(BeEmpty())
		})

		// Cleaning up on data-plane side because we have skipped the regular deletion workflow taking care of this.
		It("can be cleaned up manually", func() {
			// (1) Delete the Central CR.
			centralRef := &v1alpha1.Central{
				ObjectMeta: metav1.ObjectMeta{
					Name:      centralRequestName,
					Namespace: namespaceName,
				},
			}
			Expect(k8sClient.Delete(ctx, centralRef)).ToNot(HaveOccurred())

			// (2) Delete the namespace and everything in it.
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
			}
			Expect(k8sClient.Delete(ctx, namespace)).ToNot(HaveOccurred())
		})

		It("should delete external DNS entries", func() {
			SkipIf(!dnsEnabled, skipDNSMsg)
			var centralRequest public.CentralRequest
			Expect(obtainCentralRequest(ctx, client, centralRequestID, &centralRequest)).
				To(Succeed())
			dnsRecordsLoader := dns.NewRecordsLoader(route53Client, centralRequest)
			Eventually(dnsRecordsLoader.LoadDNSRecords).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(BeEmpty(), "Started at %s", time.Now())
		})
	})
})

func printNotes(notes []string) {
	for _, note := range notes {
		GinkgoWriter.Println(note)
	}
}
