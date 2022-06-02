package e2e

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	v1 "k8s.io/api/core/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var centralName = fmt.Sprintf("%s-%d", "e2e-test-central", time.Now().UnixMilli())

var _ = Describe("Central creation", func() {
	var _ = Describe("new", func() {
		Describe("should be created and deployed to k8s", func() {
			client, err := fleetmanager.NewClient("http://localhost:8000", "cluster-id")
			Expect(err).To(BeNil())

			request := public.CentralRequestPayload{
				Name:          centralName,
				MultiAz:       true,
				CloudProvider: "standalone",
				Region:        "standalone",
			}

			var createdCentral *public.CentralRequest
			It("created a central", func() {
				createdCentral, err = client.CreateCentral(request)
				Expect(err).To(BeNil())
				if err != nil {
					AbortSuite("Error while creating Central")
				}
				Expect(constants.DinosaurRequestStatusAccepted.String()).To(Equal(createdCentral.Status))
			})

			It("should transition central's state to provisioning", func() {
				Eventually(func() string {
					provisioningCentral, err := client.GetCentral(createdCentral.Id)
					Expect(err).To(BeNil())
					return provisioningCentral.Status
				}, 1*time.Minute).Should(Equal(constants.DinosaurRequestStatusProvisioning.String()))
			})

			It("should create central namespace", func() {
				Eventually(func() string {
					ns := &v1.Namespace{}
					err := k8sClient.Get(context.Background(), ctrlClient.ObjectKey{Name: centralName}, ns)
					Expect(err).To(BeNil())
					fmt.Println("BLAAAAAA")
					return ns.GetName()
				}).WithTimeout(5 * time.Minute).WithPolling(1 * time.Second).Should(Equal(centralName))
			})

			It("should create central in managed cluster", func() {
				Eventually(func() string {
					central := &v1alpha1.Central{}
					err := k8sClient.Get(context.Background(), ctrlClient.ObjectKey{Name: centralName, Namespace: centralName}, central)
					Expect(err).To(BeNil())
					return central.GetName()
				}, 5*time.Minute, 1*time.Second).Should(Equal(centralName))
			})

			//TODO: Add test to eventually reach ready state
		})
	})
})
