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

// TODO(create-ticket): Why is a central always created as a "eval" instance type?
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
				Expect(constants.DinosaurRequestStatusAccepted.String()).To(Equal(createdCentral.Status))
			})

			It("should transition central's state to provisioning", func() {
				Eventually(func() string {
					provisioningCentral, err := client.GetCentral(createdCentral.Id)
					Expect(err).To(BeNil())
					return provisioningCentral.Status
				}).WithTimeout(2 * time.Minute).Should(Equal(constants.DinosaurRequestStatusProvisioning.String()))
			})

			It("should create central namespace", func() {
				Eventually(func() error {
					ns := &v1.Namespace{}
					return k8sClient.Get(context.Background(), ctrlClient.ObjectKey{Name: centralName}, ns)
				}).WithTimeout(5 * time.Minute).WithPolling(1 * time.Second).Should(Succeed())
			})

			It("should create central in managed cluster", func() {
				Eventually(func() error {
					central := &v1alpha1.Central{}
					return k8sClient.Get(context.Background(), ctrlClient.ObjectKey{Name: centralName, Namespace: centralName}, central)
				}).WithTimeout(5 * time.Minute).WithPolling(1 * time.Second).Should(Succeed())
			})

			//TODO(create-ticket): Add test to eventually reach ready state
		})
	})
})
