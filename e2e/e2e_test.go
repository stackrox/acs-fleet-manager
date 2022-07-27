package e2e

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO(create-ticket): Why is a central always created as a "eval" instance type?
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
)

// TODO(create-ticket): Use correct OCM_TOKEN for different clients (console.redhat.com, fleetshard)
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

		adminClient, err = NewAdminClient(fleetManagerEndpoint, auth)
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
		// TODO(create-ticket): Create test to check Central is correctly exposed

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

	})

	Describe("should be created and deployed to k8s with admin API", func() {
		var err error
		centralName := newCentralName()

		centralResources := public.ResourceRequirements{
			Requests: public.ResourceList{
				Cpu: "501m", Memory: "201M",
			},
			Limits: public.ResourceList{
				Cpu: "502m", Memory: "202M",
			},
		}
		centralSpec := public.CentralSpec{
			Resources: centralResources,
		}
		scannerResources := public.ResourceRequirements{
			Requests: public.ResourceList{
				Cpu: "301m", Memory: "151M",
			},
			Limits: public.ResourceList{
				Cpu: "302m", Memory: "152M",
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
		It("created a central with custom resource configuration", func() {
			createdCentral, err = adminClient.CreateCentral(request)
			Expect(err).To(BeNil())
			namespaceName, err = services.FormatNamespace(createdCentral.Id)
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
			expectedResources, err := api.ConvertPublicResourceRequirementsToCoreV1(&centralResources)
			Expect(err).ToNot(HaveOccurred())
			Expect(coreV1Resources).To(Equal(expectedResources))
		})

		It("scanner analyzer resources match configured settings", func() {
			coreV1Resources := central.Spec.Scanner.Analyzer.DeploymentSpec.Resources
			expectedResources, err := api.ConvertPublicResourceRequirementsToCoreV1(&scannerResources)
			Expect(err).ToNot(HaveOccurred())
			Expect(coreV1Resources).To(Equal(expectedResources))

			a := central.Spec.Scanner.Analyzer.Scaling
			b, err := api.ConvertPublicScalingToV1(&scannerScaling)
			Expect(err).ToNot(HaveOccurred())
			Expect(a).To(Equal(b))
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

	})

})

func centralStatus(createdCentral *public.CentralRequest, client *fleetmanager.Client) string {
	Expect(createdCentral).NotTo(BeNil())
	central, err := client.GetCentral(createdCentral.Id)
	Expect(err).To(BeNil())
	return central.Status
}
