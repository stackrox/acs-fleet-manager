package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
)

// TODO(create-ticket): Why is a central always created as a "eval" instance type?
var centralName = fmt.Sprintf("%s-%d", "e2e-test-central", time.Now().UnixMilli())

// TODO(create-ticket): Use correct OCM_TOKEN for different clients (console.redhat.com, fleetshard)
var _ = Describe("Central", func() {
	Describe("should be created and deployed to k8s", func() {
		client, err := fleetmanager.NewClient("http://localhost:8000", "cluster-id")
		Expect(err).To(BeNil())

		request := public.CentralRequestPayload{
			Name:          centralName,
			MultiAz:       true,
			CloudProvider: dpCloudProvider,
			Region:        dpRegion,
		}

		var createdCentral *public.CentralRequest
		It("created a central", func() {
			createdCentral, err = client.CreateCentral(request)
			Expect(err).To(BeNil())
			Expect(constants.DinosaurRequestStatusAccepted.String()).To(Equal(createdCentral.Status))
		})

		It("should transition central's state to provisioning", func() {
			Eventually(func() string {
				Expect(createdCentral).NotTo(BeNil())
				provisioningCentral, err := client.GetCentral(createdCentral.Id)
				Expect(err).To(BeNil())
				return provisioningCentral.Status
			}).WithTimeout(waitTimeout).Should(Equal(constants.DinosaurRequestStatusProvisioning.String()))
		})

		/*
		//TODO(create-ticket): fails because the namespace is not centralName anymore but `formatNamespace(dinosaurRequest.ID)`
		// and that is not accessible from a value `*public.CentralRequest`
		It("should create central namespace", func() {
			Eventually(func() error {
				ns := &v1.Namespace{}
				return k8sClient.Get(context.Background(), ctrlClient.ObjectKey{Name: centralName}, ns)
			}).WithTimeout(waitTimeout).WithPolling(1 * time.Second).Should(Succeed())
		})

		It("should create central in its namespace on a managed cluster", func() {
			Eventually(func() error {
				central := &v1alpha1.Central{}
				return k8sClient.Get(context.Background(), ctrlClient.ObjectKey{Name: centralName, Namespace: centralName}, central)
			}).WithTimeout(waitTimeout).WithPolling(1 * time.Second).Should(Succeed())
		})
		*/
		//TODO(create-ticket): Add test to eventually reach ready state
		//TODO(yury): Add removal test
	})
})
