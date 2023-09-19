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
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
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

var _ = Describe("Central", func() {
	var client *fleetmanager.Client
	var adminAPI fleetmanager.AdminAPI
	var notes []string

	BeforeEach(func() {
		if !runCentralTests {
			Skip("Skipping Central tests")
		}

		option := fleetmanager.OptionFromEnv()
		auth, err := fleetmanager.NewStaticAuth(fleetmanager.StaticOption{StaticToken: option.Static.StaticToken})
		Expect(err).ToNot(HaveOccurred())
		client, err = fleetmanager.NewClient(fleetManagerEndpoint, auth)
		Expect(err).ToNot(HaveOccurred())

		adminStaticToken := os.Getenv("STATIC_TOKEN_ADMIN")
		adminAuth, err := fleetmanager.NewStaticAuth(fleetmanager.StaticOption{StaticToken: adminStaticToken})
		Expect(err).ToNot(HaveOccurred())
		adminClient, err := fleetmanager.NewClient(fleetManagerEndpoint, adminAuth)
		adminAPI = adminClient.AdminAPI()
		Expect(err).ToNot(HaveOccurred())

		GinkgoWriter.Printf("Current time: %s\n", time.Now().String())
		printNotes(notes)
	})

	Describe("should be created and deployed to k8s", func() {
		var err error
		var createdCentral *public.CentralRequest
		var namespaceName string

		It("created a central", func() {
			centralName := newCentralName()
			request := public.CentralRequestPayload{
				CloudProvider: dpCloudProvider,
				MultiAz:       true,
				Name:          centralName,
				Region:        dpRegion,
			}
			resp, _, err := client.PublicAPI().CreateCentral(context.Background(), true, request)
			Expect(err).To(BeNil())
			createdCentral = &resp
			notes = []string{
				fmt.Sprintf("Central name: %s", createdCentral.Name),
				fmt.Sprintf("Central ID: %s", createdCentral.Id),
			}
			printNotes(notes)
			namespaceName, err = services.FormatNamespace(createdCentral.Id)
			Expect(err).To(BeNil())
			Expect(constants.CentralRequestStatusAccepted.String()).To(Equal(createdCentral.Status))
		})

		It("should transition central's state to provisioning", func() {
			if createdCentral == nil {
				Fail("central not created")
			}
			Eventually(func() string {
				return centralStatus(createdCentral.Id, client)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Equal(constants.CentralRequestStatusProvisioning.String()))
		})

		ns := &corev1.Namespace{}
		It("should create central namespace", func() {
			if createdCentral == nil {
				Fail("central not created")
			}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), ctrlClient.ObjectKey{Name: namespaceName}, ns)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Succeed())
		})

		It("tenant namespace is labelled as a tenant namespace", func() {
			_, tenantLabelFound := ns.Labels["rhacs.redhat.com/tenant"]
			Expect(tenantLabelFound).To(BeTrue())
		})

		It("should create central in its namespace on a managed cluster", func() {
			if createdCentral == nil {
				Fail("central not created")
			}
			Eventually(func() error {
				central := &v1alpha1.Central{}
				return k8sClient.Get(context.Background(), ctrlClient.ObjectKey{Name: createdCentral.Name, Namespace: namespaceName}, central)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Succeed())
		})

		It("should not expose URLs until the routes are created", func() {
			if createdCentral == nil {
				Fail("central not created")
			}
			if !routesEnabled {
				Skip(skipRouteMsg)
			}
			Expect(createdCentral.CentralUIURL).To(BeEmpty())
			Expect(createdCentral.CentralDataURL).To(BeEmpty())
		})

		It("should transition central's state to ready", func() {
			if createdCentral == nil {
				Fail("central not created")
			}
			Eventually(func() string {
				return centralStatus(createdCentral.Id, client)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Equal(constants.CentralRequestStatusReady.String()))
		})

		It("should create central routes", func() {
			if createdCentral == nil {
				Fail("central not created")
			}
			if !routesEnabled {
				Skip(skipRouteMsg)
			}

			central := getCentral(createdCentral.Id, client)

			var reencryptRoute *openshiftRouteV1.Route
			Eventually(func() error {
				reencryptRoute, err = routeService.FindReencryptRoute(context.Background(), namespaceName)
				if err != nil {
					return fmt.Errorf("failed finding reencrypt route: %v", err)
				}
				return nil
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Succeed())

			centralUIURL, err := url.Parse(central.CentralUIURL)
			Expect(err).ToNot(HaveOccurred())
			Expect(centralUIURL.Scheme).To(Equal("https"))
			Expect(reencryptRoute.Spec.Host).To(Equal(centralUIURL.Host))
			Expect(reencryptRoute.Spec.TLS.Termination).To(Equal(openshiftRouteV1.TLSTerminationReencrypt))

			var passthroughRoute *openshiftRouteV1.Route
			Eventually(func() error {
				passthroughRoute, err = routeService.FindPassthroughRoute(context.Background(), namespaceName)
				if err != nil {
					return fmt.Errorf("failed finding passthrough route: %v", err)
				}
				return nil
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Succeed())

			centralDataHost, centralDataPort, err := net.SplitHostPort(central.CentralDataURL)
			Expect(err).ToNot(HaveOccurred())
			Expect(passthroughRoute.Spec.Host).To(Equal(centralDataHost))
			Expect(centralDataPort).To(Equal("443"))
			Expect(passthroughRoute.Spec.TLS.Termination).To(Equal(openshiftRouteV1.TLSTerminationPassthrough))
		})

		It("should create AWS Route53 records", func() {
			if createdCentral == nil {
				Fail("central not created")
			}
			if !dnsEnabled {
				Skip(skipDNSMsg)
			}

			central := getCentral(createdCentral.Id, client)
			var reencryptIngress *openshiftRouteV1.RouteIngress
			Eventually(func() error {
				reencryptIngress, err = routeService.FindReencryptIngress(context.Background(), namespaceName)
				if err != nil {
					return fmt.Errorf("failed finding reencrypt ingress: %v", err)
				}
				return nil
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Succeed())
			Expect(err).ToNot(HaveOccurred())
			dnsRecordsLoader := dns.NewRecordsLoader(route53Client, central)

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
			if createdCentral == nil {
				Fail("central not created")
			}
			var expected int32 = 3
			Eventually(func() error {
				var egressProxyDeployment appsv1.Deployment
				key := ctrlClient.ObjectKey{Namespace: namespaceName, Name: "egress-proxy"}
				if err := k8sClient.Get(context.TODO(), key, &egressProxyDeployment); err != nil {
					return err
				}
				if egressProxyDeployment.Status.ReadyReplicas < expected {
					statusBytes, _ := yaml.Marshal(&egressProxyDeployment.Status)
					return fmt.Errorf("egress proxy only has %d/%d ready replicas (and %d unavailable ones), expected %d. full status: %s", egressProxyDeployment.Status.ReadyReplicas, egressProxyDeployment.Status.Replicas, egressProxyDeployment.Status.UnavailableReplicas, expected, statusBytes)
				}
				return nil
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Succeed())
		})

		// TODO(ROX-11368): Add test to eventually reach ready state
		// TODO(ROX-11368): create test to check that Central and Scanner are healthy
		// TODO(ROX-11368): Create test to check Central is correctly exposed

		It("should transition central to deprovisioning state", func() {
			if createdCentral == nil {
				Fail("central not created")
			}
			_, err = client.PublicAPI().DeleteCentralById(context.TODO(), createdCentral.Id, true)
			Expect(err).To(Succeed())
			Eventually(func() string {
				return centralStatus(createdCentral.Id, client)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Equal(constants.CentralRequestStatusDeprovision.String()))
		})

		It("should delete central CR", func() {
			if createdCentral == nil {
				Fail("central not created")
			}
			Eventually(func() bool {
				central := &v1alpha1.Central{}
				err := k8sClient.Get(context.TODO(), ctrlClient.ObjectKey{Name: createdCentral.Name, Namespace: namespaceName}, central)
				return apiErrors.IsNotFound(err)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(BeTrue())
		})

		It("should delete the egress proxy", func() {
			if createdCentral == nil {
				Fail("central not created")
			}
			Eventually(func() error {
				var egressProxyDeployment appsv1.Deployment
				key := ctrlClient.ObjectKey{Namespace: namespaceName, Name: "egress-proxy"}
				return k8sClient.Get(context.TODO(), key, &egressProxyDeployment)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Satisfy(apiErrors.IsNotFound))
		})

		It("should remove central namespace", func() {
			if createdCentral == nil {
				Fail("central not created")
			}
			Eventually(func() bool {
				ns := &corev1.Namespace{}
				err := k8sClient.Get(context.Background(), ctrlClient.ObjectKey{Name: namespaceName}, ns)
				return apiErrors.IsNotFound(err)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(BeTrue())
		})

		It("should delete external DNS entries", func() {
			if createdCentral == nil {
				Fail("central not created")
			}
			if !dnsEnabled {
				Skip(skipDNSMsg)
			}

			central := getCentral(createdCentral.Id, client)
			dnsRecordsLoader := dns.NewRecordsLoader(route53Client, central)

			Eventually(dnsRecordsLoader.LoadDNSRecords).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(BeEmpty(), "Started at %s", time.Now())
		})
	})

	Describe("should be created and deployed to k8s with admin API", func() {
		var err error
		var centralID string
		var createdCentral *private.CentralRequest
		var namespaceName string

		It("should create central", func() {
			centralName := newCentralName()
			request := private.CentralRequestPayload{
				Name:          centralName,
				MultiAz:       true,
				CloudProvider: dpCloudProvider,
				Region:        dpRegion,
			}
			resp, _, err := adminAPI.CreateCentral(context.TODO(), true, request)
			Expect(err).To(BeNil())
			createdCentral = &resp
			notes = []string{
				fmt.Sprintf("Central name: %s", createdCentral.Name),
				fmt.Sprintf("Central ID: %s", createdCentral.Id),
			}
			centralID = createdCentral.Id
			namespaceName, err = services.FormatNamespace(centralID)
			Expect(err).To(BeNil())
			Expect(constants.CentralRequestStatusAccepted.String()).To(Equal(createdCentral.Status))
		})

		central := &v1alpha1.Central{}
		It("should create central in its namespace on a managed cluster", func() {
			if createdCentral == nil {
				Fail("central not created")
			}
			Eventually(func() error {
				return k8sClient.Get(context.TODO(), ctrlClient.ObjectKey{Name: createdCentral.Name, Namespace: namespaceName}, central)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Succeed())
		})

		It("should transition central's state to ready", func() {
			if createdCentral == nil {
				Fail("central not created")
			}
			Eventually(func() string {
				return centralStatus(createdCentral.Id, client)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Equal(constants.CentralRequestStatusReady.String()))
		})

		It("should transition central to deprovisioning state", func() {
			if createdCentral == nil {
				Fail("central not created")
			}
			_, err = client.PublicAPI().DeleteCentralById(context.TODO(), createdCentral.Id, true)
			Expect(err).To(Succeed())
			Eventually(func() string {
				return centralStatus(createdCentral.Id, client)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Equal(constants.CentralRequestStatusDeprovision.String()))
		})

		It("should delete central CR", func() {
			if createdCentral == nil {
				Fail("central not created")
			}
			Eventually(func() bool {
				central := &v1alpha1.Central{}
				err := k8sClient.Get(context.Background(), ctrlClient.ObjectKey{Name: createdCentral.Name, Namespace: namespaceName}, central)
				return apiErrors.IsNotFound(err)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(BeTrue())
		})

		It("should remove central namespace", func() {
			if createdCentral == nil {
				Fail("central not created")
			}
			Eventually(func() bool {
				ns := &corev1.Namespace{}
				err := k8sClient.Get(context.Background(), ctrlClient.ObjectKey{Name: namespaceName}, ns)
				return apiErrors.IsNotFound(err)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(BeTrue())
		})

		It("should delete external DNS entries", func() {
			if createdCentral == nil {
				Fail("central not created")
			}
			if !dnsEnabled {
				Skip(skipDNSMsg)
			}

			central := getCentral(createdCentral.Id, client)
			dnsRecordsLoader := dns.NewRecordsLoader(route53Client, central)

			Eventually(dnsRecordsLoader.LoadDNSRecords).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(BeEmpty(), "Started at %s", time.Now())
		})

	})

	Describe("should be deployed and can be force-deleted", func() {
		var err error
		var createdCentral *public.CentralRequest
		var central *public.CentralRequest
		var namespaceName string

		It("created a central", func() {
			centralName := newCentralName()
			request := public.CentralRequestPayload{
				Name:          centralName,
				MultiAz:       true,
				CloudProvider: dpCloudProvider,
				Region:        dpRegion,
			}

			resp, _, err := client.PublicAPI().CreateCentral(context.TODO(), true, request)
			Expect(err).To(BeNil())
			createdCentral = &resp
			notes = []string{
				fmt.Sprintf("Central name: %s", createdCentral.Name),
				fmt.Sprintf("Central ID: %s", createdCentral.Id),
			}
			namespaceName, err = services.FormatNamespace(createdCentral.Id)
			Expect(err).To(BeNil())
			Expect(constants.CentralRequestStatusAccepted.String()).To(Equal(createdCentral.Status))
		})

		It("should transition central's state to ready", func() {
			if createdCentral == nil {
				Fail("central not created")
			}
			Eventually(func() string {
				return centralStatus(createdCentral.Id, client)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Equal(constants.CentralRequestStatusReady.String()))
			central = getCentral(createdCentral.Id, client)
		})

		It("should be deletable in the control-plane database", func() {
			if createdCentral == nil {
				Fail("central not created")
			}
			_, err = adminAPI.DeleteDbCentralById(context.TODO(), createdCentral.Id)
			Expect(err).ToNot(HaveOccurred())
			_, err = adminAPI.DeleteDbCentralById(context.TODO(), createdCentral.Id)
			Expect(err).To(HaveOccurred())
			central, _, err := client.PublicAPI().GetCentralById(context.TODO(), createdCentral.Id)
			Expect(err).To(HaveOccurred())
			Expect(central.Id).To(BeEmpty())
		})

		// Cleaning up on data-plane side because we have skipped the regular deletion workflow taking care of this.
		It("can be cleaned up manually", func() {
			if createdCentral == nil {
				Fail("central not created")
			}
			// (1) Delete the Central CR.
			centralRef := &v1alpha1.Central{
				ObjectMeta: metav1.ObjectMeta{
					Name:      createdCentral.Name,
					Namespace: namespaceName,
				},
			}
			err = k8sClient.Delete(context.TODO(), centralRef)
			Expect(err).ToNot(HaveOccurred())

			// (2) Delete the namespace and everything in it.
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
			}
			err = k8sClient.Delete(context.TODO(), namespace)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should delete external DNS entries", func() {
			if createdCentral == nil {
				Fail("central not created")
			}
			if !dnsEnabled {
				Skip(skipDNSMsg)
			}

			dnsRecordsLoader := dns.NewRecordsLoader(route53Client, central)

			Eventually(dnsRecordsLoader.LoadDNSRecords).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(BeEmpty(), "Started at %s", time.Now())
		})
	})
})

func getCentral(id string, client *fleetmanager.Client) *public.CentralRequest {
	Expect(id).NotTo(BeEmpty())
	central, _, err := client.PublicAPI().GetCentralById(context.TODO(), id)
	Expect(err).To(BeNil())
	Expect(central).ToNot(BeNil())
	return &central
}

func centralStatus(id string, client *fleetmanager.Client) string {
	return getCentral(id, client).Status
}

func printNotes(notes []string) {
	for _, note := range notes {
		GinkgoWriter.Println(note)
	}
}
