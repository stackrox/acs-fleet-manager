package e2e

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/route53"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO(ROX-11468): Why is a central always created as a "eval" instance type?
var (
	centralName = fmt.Sprintf("%s-%d", "e2e-test-central", time.Now().UnixMilli())
)

const (
	defaultPolling = 1 * time.Second
	skipRouteMsg   = "route resource is not known to test cluster"
	skipDNSMsg     = "external DNS is not enabled for this test run"
)

// TODO(ROX-11465): Use correct OCM_TOKEN for different clients (console.redhat.com, fleetshard)
var _ = Describe("Central", func() {
	var client *fleetmanager.Client

	BeforeEach(func() {
		authType := "OCM"
		if val := os.Getenv("AUTH_TYPE"); val != "" {
			authType = val
		}
		fleetManagerEndpoint := "http://localhost:8000"
		if fmEndpointEnv := os.Getenv("FLEET_MANAGER_ENDPOINT"); fmEndpointEnv != "" {
			fleetManagerEndpoint = fmEndpointEnv
		}

		auth, err := fleetmanager.NewAuth(authType)
		Expect(err).ToNot(HaveOccurred())
		client, err = fleetmanager.NewClient(fleetManagerEndpoint, "cluster-id", auth)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("should be created and deployed to k8s", func() {
		var err error

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

		// TODO(create-ticket): Add test for passthrough route created by fleetshard-sync
		It("should create central routes", func() {
			if !routesEnabled {
				Skip(skipRouteMsg)
			}

			central := getCentral(createdCentral, client)

			var reencryptRoute *openshiftRouteV1.Route
			Eventually(func() error {
				reencryptRoute, err = routeService.FindReencryptRoute(context.Background(), namespaceName)
				return err
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Succeed())

			Expect(reencryptRoute.Spec.Host).To(Equal(central.Host))
			Expect(reencryptRoute.Spec.TLS.Termination).To(Equal(openshiftRouteV1.TLSTerminationReencrypt))
		})

		It("should create AWS Route53 records", func() {
			if !dnsEnabled {
				Skip(skipDNSMsg)
			}

			central := getCentral(createdCentral, client)
			routeCanonicalName, err := routeService.FindReencryptCanonicalHostname(context.Background(), namespaceName)
			Expect(err).ToNot(HaveOccurred())

			rhacsZone, err := getHostedZone(central)
			Expect(err).ToNot(HaveOccurred())
			Expect(rhacsZone).ToNot(BeNil())

			var records *route53.ListResourceRecordSetsOutput
			Eventually(func() int {
				records, err = route53Client.ListResourceRecordSets(&route53.ListResourceRecordSetsInput{
					HostedZoneId:    rhacsZone.Id,
					StartRecordName: &central.Host,
				})
				Expect(err).ToNot(HaveOccurred())
				return len(records.ResourceRecordSets)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Equal(1))

			Expect(len(records.ResourceRecordSets)).To(Equal(1))
			recordSet := records.ResourceRecordSets[0]
			Expect(len(recordSet.ResourceRecords)).To(Equal(1))
			record := recordSet.ResourceRecords[0]

			name := removeLastChar(*recordSet.Name)
			Expect(name).To(Equal(central.Host))
			Expect(*record.Value).To(Equal(routeCanonicalName))

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

			rhacsZone, err := getHostedZone(central)
			Expect(err).ToNot(HaveOccurred())
			Expect(rhacsZone).ToNot(BeNil())

			var records *route53.ListResourceRecordSetsOutput
			Eventually(func() int {
				records, err = route53Client.ListResourceRecordSets(&route53.ListResourceRecordSetsInput{
					HostedZoneId:    rhacsZone.Id,
					StartRecordName: &central.Host,
				})
				Expect(err).ToNot(HaveOccurred())
				return len(records.ResourceRecordSets)
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Equal(0))
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

func isDNSEnabled(routesEnabled bool) bool {
	hasRoute53Creds := os.Getenv("ROUTE53_ACCESS_KEY") != "" &&
		os.Getenv("ROUTE53_SECRET_ACCESS_KEY") != "" &&
		os.Getenv("ENABLE_DINOSAUR_EXTERNAL_CERTIFICATE") != ""
	return hasRoute53Creds && routesEnabled
}

func removeLastChar(s string) string {
	return s[:len(s)-1]
}

func getHostedZone(central *public.CentralRequest) (*route53.HostedZone, error) {
	hostedZones, err := route53Client.ListHostedZones(&route53.ListHostedZonesInput{})
	if err != nil {
		return nil, err
	}

	var rhacsZone *route53.HostedZone
	for _, zone := range hostedZones.HostedZones {
		// Omit the . at the end of hosted zone name
		name := removeLastChar(*zone.Name)
		if strings.Contains(central.Host, name) {
			rhacsZone = zone
			break
		}
	}

	return rhacsZone, nil
}
