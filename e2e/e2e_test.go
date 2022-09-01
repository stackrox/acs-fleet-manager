package e2e

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/stackrox/acs-fleet-manager/e2e/dns"
	"github.com/stackrox/acs-fleet-manager/e2e/envtokenauth"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/converters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
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

// TODO(ROX-11465): Use correct OCM_TOKEN for different clients (console.redhat.com, fleetshard)
var _ = Describe("Central", func() {
	var client *fleetmanager.Client
	var adminClient *Client

	BeforeEach(func() {
		authType := "OCM"
		if val := os.Getenv("AUTH_TYPE"); val != "" {
			authType = val
		}
		GinkgoWriter.Printf("AUTH_TYPE=%q\n", authType)

		fleetManagerEndpoint := "http://localhost:8000"
		if fmEndpointEnv := os.Getenv("FLEET_MANAGER_ENDPOINT"); fmEndpointEnv != "" {
			fleetManagerEndpoint = fmEndpointEnv
		}
		GinkgoWriter.Printf("FLEET_MANAGER_ENDPOINT=%q\n", fleetManagerEndpoint)

		auth, err := fleetmanager.NewAuth(authType)
		Expect(err).ToNot(HaveOccurred())
		client, err = fleetmanager.NewClient(fleetManagerEndpoint, "cluster-id", auth)
		Expect(err).ToNot(HaveOccurred())

		adminAuth, err := envtokenauth.CreateAuth("STATIC_TOKEN_ADMIN")
		Expect(err).ToNot(HaveOccurred())
		adminClient, err = NewAdminClient(fleetManagerEndpoint, adminAuth)
		Expect(err).ToNot(HaveOccurred())

	})

	Describe("should be created and deployed to k8s", func() {
		var err error

		centralName := newCentralName()
		request := public.CentralRequestPayload{
			Name:          centralName,
			MultiAz:       true,
			CloudProvider: dpCloudProvider,
			Region:        dpRegion,
		}

		var createdCentral *public.CentralRequest
		var namespaceName string
		It("created a central", func() {
			createdCentral, err = client.CreateCentral(request)
			Expect(err).To(BeNil())
			namespaceName, err = services.FormatNamespace(createdCentral.Id)
			Expect(err).To(BeNil())
			Expect(constants.DinosaurRequestStatusAccepted.String()).To(Equal(createdCentral.Status))
		})

		It("should transition central's state to provisioning", func() {
			Eventually(func() string {
				return centralStatus(createdCentral, client)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Equal(constants.DinosaurRequestStatusProvisioning.String()))
		})

		// TODO(create-ticket): fails because the namespace is not centralName anymore but `formatNamespace(dinosaurRequest.ID)`
		// and that is not accessible from a value `*public.CentralRequest`
		It("should create central namespace", func() {
			Eventually(func() error {
				ns := &v1.Namespace{}
				return k8sClient.Get(context.Background(), ctrlClient.ObjectKey{Name: namespaceName}, ns)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Succeed())
		})

		It("should create central in its namespace on a managed cluster", func() {
			Eventually(func() error {
				central := &v1alpha1.Central{}
				return k8sClient.Get(context.Background(), ctrlClient.ObjectKey{Name: centralName, Namespace: namespaceName}, central)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Succeed())
		})

		// TODO(create-ticket): create test to check that Central and Scanner are healthy
		It("should create central routes", func() {
			if !routesEnabled {
				Skip(skipRouteMsg)
			}

			central := getCentral(createdCentral, client)

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

			centralDataURL, err := url.Parse(central.CentralDataURL)
			Expect(err).ToNot(HaveOccurred())
			Expect(centralDataURL.Scheme).To(Equal("https"))
			Expect(passthroughRoute.Spec.Host).To(Equal(centralDataURL.Host))
			Expect(passthroughRoute.Spec.TLS.Termination).To(Equal(openshiftRouteV1.TLSTerminationPassthrough))
		})

		It("should create AWS Route53 records", func() {
			if !dnsEnabled {
				Skip(skipDNSMsg)
			}

			central := getCentral(createdCentral, client)
			reencryptIngress, err := routeService.FindReencryptIngress(context.Background(), namespaceName)
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

		It("should transition central's state to ready", func() {
			Eventually(func() string {
				return centralStatus(createdCentral, client)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Equal(constants.DinosaurRequestStatusReady.String()))
		})
		// TODO(ROX-11368): Add test to eventually reach ready state
		// TODO(ROX-11368): create test to check that Central and Scanner are healthy
		// TODO(ROX-11368): Create test to check Central is correctly exposed

		It("should transition central to deprovisioning state", func() {
			err = client.DeleteCentral(createdCentral.Id)
			Expect(err).To(Succeed())
			Eventually(func() string {
				deprovisioningCentral, err := client.GetCentral(createdCentral.Id)
				Expect(err).To(BeNil())
				return deprovisioningCentral.Status
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Equal(constants.DinosaurRequestStatusDeprovision.String()))
		})

		It("should delete central CR", func() {
			Eventually(func() bool {
				central := &v1alpha1.Central{}
				err := k8sClient.Get(context.Background(), ctrlClient.ObjectKey{Name: centralName, Namespace: centralName}, central)
				return apiErrors.IsNotFound(err)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(BeTrue())
		})

		It("should remove central namespace", func() {
			Eventually(func() bool {
				ns := &v1.Namespace{}
				err := k8sClient.Get(context.Background(), ctrlClient.ObjectKey{Name: namespaceName}, ns)
				return apiErrors.IsNotFound(err)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(BeTrue())
		})

		It("should delete external DNS entries", func() {
			if !dnsEnabled {
				Skip(skipDNSMsg)
			}

			central := getCentral(createdCentral, client)
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
		centralName := newCentralName()

		centralResources := public.ResourceRequirements{
			Requests: map[string]string{
				v1.ResourceCPU.String():    "501m",
				v1.ResourceMemory.String(): "201M",
			},
			Limits: map[string]string{
				v1.ResourceCPU.String():    "502m",
				v1.ResourceMemory.String(): "202M",
			},
		}
		centralSpec := public.CentralSpec{
			Resources: centralResources,
		}
		scannerResources := public.ResourceRequirements{
			Requests: map[string]string{
				v1.ResourceCPU.String():    "301m",
				v1.ResourceMemory.String(): "151M",
			},
			Limits: map[string]string{
				v1.ResourceCPU.String():    "302m",
				v1.ResourceMemory.String(): "152M",
			},
		}
		scannerScaling := public.ScannerSpecAnalyzerScaling{
			AutoScaling: "Enabled",
			Replicas:    1,
			MinReplicas: 1,
			MaxReplicas: 2,
		}
		scannerSpec := public.ScannerSpec{
			Analyzer: public.ScannerSpecAnalyzer{
				Resources: scannerResources,
				Scaling:   scannerScaling,
			},
		}
		request := public.CentralRequestPayload{
			Name:          centralName,
			MultiAz:       true,
			CloudProvider: dpCloudProvider,
			Region:        dpRegion,
			Central:       centralSpec,
			Scanner:       scannerSpec,
		}

		var createdCentral *public.CentralRequest
		var namespaceName string
		It("should create central with custom resource configuration", func() {
			createdCentral, err = adminClient.CreateCentral(request)
			Expect(err).To(BeNil())
			centralID = createdCentral.Id
			namespaceName, err = services.FormatNamespace(centralID)
			Expect(err).To(BeNil())
			Expect(constants.DinosaurRequestStatusAccepted.String()).To(Equal(createdCentral.Status))
		})

		central := &v1alpha1.Central{}
		It("should create central in its namespace on a managed cluster", func() {
			Eventually(func() error {
				return k8sClient.Get(context.Background(), ctrlClient.ObjectKey{Name: centralName, Namespace: namespaceName}, central)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Succeed())
		})

		It("central resources match configured settings", func() {
			coreV1Resources := central.Spec.Central.DeploymentSpec.Resources
			expectedResources, err := converters.ConvertPublicResourceRequirementsToCoreV1(&centralResources)
			Expect(err).ToNot(HaveOccurred())
			Expect(*coreV1Resources).To(Equal(expectedResources))
		})

		It("scanner analyzer resources match configured settings", func() {
			coreV1Resources := central.Spec.Scanner.Analyzer.DeploymentSpec.Resources
			expectedResources, err := converters.ConvertPublicResourceRequirementsToCoreV1(&scannerResources)
			Expect(err).ToNot(HaveOccurred())
			Expect(*coreV1Resources).To(Equal(expectedResources))

			scaling := central.Spec.Scanner.Analyzer.Scaling
			expectedScaling, err := converters.ConvertPublicScalingToV1(&scannerScaling)
			Expect(err).ToNot(HaveOccurred())
			Expect(*scaling).To(Equal(expectedScaling))
		})

		It("resources should be updatable", func() {
			updateReq := private.CentralUpdateRequest{
				Central: private.CentralSpec{
					Resources: private.ResourceRequirements{
						Requests: map[string]string{
							v1.ResourceMemory.String(): "199M",
						},
						Limits: map[string]string{
							v1.ResourceCPU.String(): "499m",
						},
					},
				},
			}
			newCentralResources := v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("501m"),
					v1.ResourceMemory: resource.MustParse("199M"),
				},
				Limits: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("499m"),
					v1.ResourceMemory: resource.MustParse("202M"),
				},
			}

			_, err = adminClient.UpdateCentral(centralID, updateReq)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() v1.ResourceRequirements {
				central := &v1alpha1.Central{}
				err := k8sClient.Get(context.Background(), ctrlClient.ObjectKey{Name: centralName, Namespace: namespaceName}, central)
				Expect(err).ToNot(HaveOccurred())
				if central.Spec.Central == nil || central.Spec.Central.Resources == nil {
					return v1.ResourceRequirements{}
				}
				return *central.Spec.Central.Resources
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Equal(newCentralResources))
		})

		It("should transition central's state to ready", func() {
			Eventually(func() string {
				return centralStatus(createdCentral, client)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Equal(constants.DinosaurRequestStatusReady.String()))
		})

		It("should transition central to deprovisioning state", func() {
			err = client.DeleteCentral(createdCentral.Id)
			Expect(err).To(Succeed())
			Eventually(func() string {
				deprovisioningCentral, err := client.GetCentral(createdCentral.Id)
				Expect(err).To(BeNil())
				return deprovisioningCentral.Status
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Equal(constants.DinosaurRequestStatusDeprovision.String()))
		})

		It("should delete central CR", func() {
			Eventually(func() bool {
				central := &v1alpha1.Central{}
				err := k8sClient.Get(context.Background(), ctrlClient.ObjectKey{Name: centralName, Namespace: centralName}, central)
				return apiErrors.IsNotFound(err)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(BeTrue())
		})

		It("should remove central namespace", func() {
			Eventually(func() bool {
				ns := &v1.Namespace{}
				err := k8sClient.Get(context.Background(), ctrlClient.ObjectKey{Name: namespaceName}, ns)
				return apiErrors.IsNotFound(err)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(BeTrue())
		})

		It("should delete external DNS entries", func() {
			if !dnsEnabled {
				Skip(skipDNSMsg)
			}

			central := getCentral(createdCentral, client)
			dnsRecordsLoader := dns.NewRecordsLoader(route53Client, central)

			Eventually(dnsRecordsLoader.LoadDNSRecords).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(BeEmpty(), "Started at %s", time.Now())
		})

	})
})

func getCentral(createdCentral *public.CentralRequest, client *fleetmanager.Client) *public.CentralRequest {
	Expect(createdCentral).NotTo(BeNil())
	central, err := client.GetCentral(createdCentral.Id)
	Expect(err).To(BeNil())
	return central
}

func centralStatus(createdCentral *public.CentralRequest, client *fleetmanager.Client) string {
	return getCentral(createdCentral, client).Status
}
