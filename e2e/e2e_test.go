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
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/converters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
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

// TODO(ROX-11465): Use correct OCM_TOKEN for different clients (console.redhat.com, fleetshard)
var _ = Describe("Central", func() {
	var client *fleetmanager.Client
	var adminAPI *private.DefaultApiService
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

		auth, err := fleetmanager.NewAuth(authType, fleetmanager.WithOptionFromEnv())
		Expect(err).ToNot(HaveOccurred())
		client, err = fleetmanager.NewClient(fleetManagerEndpoint, auth)
		Expect(err).ToNot(HaveOccurred())

		adminStaticToken := os.Getenv("STATIC_TOKEN_ADMIN")
		adminAuth, err := fleetmanager.NewAuth("STATIC_TOKEN", fleetmanager.WithStaticOption(fleetmanager.StaticOption{StaticToken: adminStaticToken}))
		Expect(err).ToNot(HaveOccurred())
		adminClient, err := fleetmanager.NewClient(fleetManagerEndpoint, adminAuth)
		adminAPI = adminClient.AdminAPI()

		Expect(err).ToNot(HaveOccurred())

	})

	Describe("should be created and deployed to k8s", func() {
		var err error

		centralName := newCentralName()
		request := public.CentralRequestPayload{
			CloudProvider: dpCloudProvider,
			MultiAz:       true,
			Name:          centralName,
			Region:        dpRegion,
		}

		var createdCentral *public.CentralRequest
		var namespaceName string
		It("created a central", func() {
			resp, _, err := client.PublicAPI().CreateCentral(context.Background(), true, request)
			createdCentral = &resp
			Expect(err).To(BeNil())
			namespaceName, err = services.FormatNamespace(createdCentral.Id)
			Expect(err).To(BeNil())
			Expect(constants.CentralRequestStatusAccepted.String()).To(Equal(createdCentral.Status))
		})

		It("should transition central's state to provisioning", func() {
			Eventually(func() string {
				return centralStatus(createdCentral.Id, client)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Equal(constants.CentralRequestStatusProvisioning.String()))
		})

		// TODO(create-ticket): fails because the namespace is not centralName anymore but `formatNamespace(dinosaurRequest.ID)`
		// and that is not accessible from a value `*public.CentralRequest`
		It("should create central namespace", func() {
			Eventually(func() error {
				ns := &corev1.Namespace{}
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

		It("should transition central's state to ready", func() {
			Eventually(func() string {
				return centralStatus(createdCentral.Id, client)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Equal(constants.CentralRequestStatusReady.String()))
		})

		It("should spin up an egress proxy with two healthy replicas", func() {
			Eventually(func() error {
				var egressProxyDeployment appsv1.Deployment
				key := ctrlClient.ObjectKey{Namespace: namespaceName, Name: "egress-proxy"}
				if err := k8sClient.Get(context.TODO(), key, &egressProxyDeployment); err != nil {
					return err
				}
				if egressProxyDeployment.Status.ReadyReplicas < 2 {
					statusBytes, _ := yaml.Marshal(&egressProxyDeployment.Status)
					return fmt.Errorf("egress proxy only has %d/%d ready replicas (and %d unavailable ones), expected 2. full status: %s", egressProxyDeployment.Status.ReadyReplicas, egressProxyDeployment.Status.Replicas, egressProxyDeployment.Status.UnavailableReplicas, statusBytes)
				}
				return nil
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Succeed())
		})

		// TODO(ROX-11368): Add test to eventually reach ready state
		// TODO(ROX-11368): create test to check that Central and Scanner are healthy
		// TODO(ROX-11368): Create test to check Central is correctly exposed

		It("should transition central to deprovisioning state", func() {
			_, err = client.PublicAPI().DeleteCentralById(context.TODO(), createdCentral.Id, true)
			Expect(err).To(Succeed())
			Eventually(func() string {
				return centralStatus(createdCentral.Id, client)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Equal(constants.CentralRequestStatusDeprovision.String()))
		})

		It("should delete central CR", func() {
			Eventually(func() bool {
				central := &v1alpha1.Central{}
				err := k8sClient.Get(context.TODO(), ctrlClient.ObjectKey{Name: centralName, Namespace: centralName}, central)
				return apiErrors.IsNotFound(err)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(BeTrue())
		})

		It("should delete the egress proxy", func() {
			Eventually(func() error {
				var egressProxyDeployment appsv1.Deployment
				key := ctrlClient.ObjectKey{Namespace: namespaceName, Name: "egress-proxy"}
				return k8sClient.Get(context.TODO(), key, &egressProxyDeployment)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Satisfy(apiErrors.IsNotFound))
		})

		It("should remove central namespace", func() {
			Eventually(func() bool {
				ns := &corev1.Namespace{}
				err := k8sClient.Get(context.Background(), ctrlClient.ObjectKey{Name: namespaceName}, ns)
				return apiErrors.IsNotFound(err)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(BeTrue())
		})

		It("should delete external DNS entries", func() {
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
		centralName := newCentralName()

		centralResources := private.ResourceRequirements{
			Requests: map[string]string{
				corev1.ResourceCPU.String():    "501m",
				corev1.ResourceMemory.String(): "201M",
			},
			Limits: map[string]string{
				corev1.ResourceCPU.String():    "502m",
				corev1.ResourceMemory.String(): "202M",
			},
		}
		centralSpec := private.CentralSpec{
			Resources: centralResources,
		}
		scannerResources := private.ResourceRequirements{
			Requests: map[string]string{
				corev1.ResourceCPU.String():    "301m",
				corev1.ResourceMemory.String(): "151M",
			},
			Limits: map[string]string{
				corev1.ResourceCPU.String():    "302m",
				corev1.ResourceMemory.String(): "152M",
			},
		}
		scannerScaling := private.ScannerSpecAnalyzerScaling{
			AutoScaling: "Enabled",
			Replicas:    1,
			MinReplicas: 1,
			MaxReplicas: 2,
		}
		scannerSpec := private.ScannerSpec{
			Analyzer: private.ScannerSpecAnalyzer{
				Resources: scannerResources,
				Scaling:   scannerScaling,
			},
		}
		request := private.CentralRequestPayload{
			Name:          centralName,
			MultiAz:       true,
			CloudProvider: dpCloudProvider,
			Region:        dpRegion,
			Central:       centralSpec,
			Scanner:       scannerSpec,
		}

		var createdCentral *private.CentralRequest
		var namespaceName string
		It("should create central with custom resource configuration", func() {
			resp, _, err := adminAPI.CreateCentral(context.TODO(), true, request)
			createdCentral = &resp
			Expect(err).To(BeNil())
			centralID = createdCentral.Id
			namespaceName, err = services.FormatNamespace(centralID)
			Expect(err).To(BeNil())
			Expect(constants.CentralRequestStatusAccepted.String()).To(Equal(createdCentral.Status))
		})

		central := &v1alpha1.Central{}
		It("should create central in its namespace on a managed cluster", func() {
			Eventually(func() error {
				return k8sClient.Get(context.TODO(), ctrlClient.ObjectKey{Name: centralName, Namespace: namespaceName}, central)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Succeed())
		})

		It("central resources match configured settings", func() {
			coreV1Resources := central.Spec.Central.DeploymentSpec.Resources
			expectedResources, err := converters.ConvertAdminPrivateRequirementsToCoreV1(&centralResources)
			Expect(err).ToNot(HaveOccurred())
			Expect(*coreV1Resources).To(Equal(expectedResources))
		})

		It("scanner analyzer resources match configured settings", func() {
			coreV1Resources := central.Spec.Scanner.Analyzer.DeploymentSpec.Resources
			expectedResources, err := converters.ConvertAdminPrivateRequirementsToCoreV1(&scannerResources)
			Expect(err).ToNot(HaveOccurred())
			Expect(*coreV1Resources).To(Equal(expectedResources))

			scaling := central.Spec.Scanner.Analyzer.Scaling
			expectedScaling, err := converters.ConvertAdminPrivateScalingToV1(&scannerScaling)
			Expect(err).ToNot(HaveOccurred())
			Expect(*scaling).To(Equal(expectedScaling))
		})

		It("resources should be updatable", func() {
			updateReq := private.CentralUpdateRequest{
				Central: private.CentralSpec{
					Resources: private.ResourceRequirements{
						Requests: map[string]string{
							corev1.ResourceMemory.String(): "199M",
						},
						Limits: map[string]string{
							corev1.ResourceCPU.String(): "505m",
						},
					},
				},
			}
			newCentralResources := corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("501m"),
					corev1.ResourceMemory: resource.MustParse("199M"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("505m"),
					corev1.ResourceMemory: resource.MustParse("202M"),
				},
			}

			_, _, err = adminAPI.UpdateCentralById(context.TODO(), centralID, updateReq)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() corev1.ResourceRequirements {
				central := &v1alpha1.Central{}
				err := k8sClient.Get(context.Background(), ctrlClient.ObjectKey{Name: centralName, Namespace: namespaceName}, central)
				Expect(err).ToNot(HaveOccurred())
				if central.Spec.Central == nil || central.Spec.Central.Resources == nil {
					return corev1.ResourceRequirements{}
				}
				return *central.Spec.Central.Resources
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Equal(newCentralResources))
		})

		It("should transition central's state to ready", func() {
			Eventually(func() string {
				return centralStatus(createdCentral.Id, client)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Equal(constants.CentralRequestStatusReady.String()))
		})

		It("should transition central to deprovisioning state", func() {
			_, err = client.PublicAPI().DeleteCentralById(context.TODO(), createdCentral.Id, true)
			Expect(err).To(Succeed())
			Eventually(func() string {
				return centralStatus(createdCentral.Id, client)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Equal(constants.CentralRequestStatusDeprovision.String()))
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
				ns := &corev1.Namespace{}
				err := k8sClient.Get(context.Background(), ctrlClient.ObjectKey{Name: namespaceName}, ns)
				return apiErrors.IsNotFound(err)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(BeTrue())
		})

		It("should delete external DNS entries", func() {
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

		centralName := newCentralName()
		request := public.CentralRequestPayload{
			Name:          centralName,
			MultiAz:       true,
			CloudProvider: dpCloudProvider,
			Region:        dpRegion,
		}

		var createdCentral *public.CentralRequest
		var central *public.CentralRequest
		var namespaceName string

		It("created a central", func() {
			resp, _, err := client.PublicAPI().CreateCentral(context.TODO(), true, request)
			Expect(err).To(BeNil())
			createdCentral = &resp
			namespaceName, err = services.FormatNamespace(createdCentral.Id)
			Expect(err).To(BeNil())
			Expect(constants.CentralRequestStatusAccepted.String()).To(Equal(createdCentral.Status))
		})

		It("should transition central's state to ready", func() {
			Eventually(func() string {
				return centralStatus(createdCentral.Id, client)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Equal(constants.CentralRequestStatusReady.String()))
			central = getCentral(createdCentral.Id, client)
		})

		It("should be deletable in the control-plane database", func() {
			_, err = adminAPI.DeleteDbCentralById(context.TODO(), createdCentral.Id)
			Expect(err).To(Succeed())
			Expect(err).To(HaveOccurred())
			central, _, err := client.PublicAPI().GetCentralById(context.TODO(), createdCentral.Id)
			Expect(err).To(HaveOccurred())
			Expect(central).To(BeNil())
		})

		// Cleaning up on data-plane side because we have skipped the regular deletion workflow taking care of this.
		It("can be cleaned up manually", func() {
			// (1) Delete the Central CR.
			centralRef := &v1alpha1.Central{
				ObjectMeta: metav1.ObjectMeta{
					Name:      centralName,
					Namespace: namespaceName,
				},
			}
			err = k8sClient.Delete(context.Background(), centralRef)
			Expect(err).ToNot(HaveOccurred())

			// (2) Delete the namespace and everything in it.
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
			}
			err = k8sClient.Delete(context.Background(), namespace)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should delete external DNS entries", func() {
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
	central, _, err := client.PublicAPI().GetCentralById(context.Background(), id)
	Expect(err).To(BeNil())
	return &central
}

func centralStatus(id string, client *fleetmanager.Client) string {
	return getCentral(id, client).Status
}
