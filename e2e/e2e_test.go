package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"os"
)

var _ = Describe("Central creation", func() {
	var _ = Describe("new", func() {
		It("should be created", func() {
			client, err := fleetmanager.NewClient("localhost:8080", os.Getenv("OCM_TOKEN"))
			Expect(err).To(BeNil())

			request := public.CentralRequestPayload{
				Name:          "e2e-test-central",
				MultiAz:       false,
				CloudProvider: "standalone",
				Region:        "standalone",
			}
			err = client.CreateCentral(request)
			Expect(err).To(BeNil())
		})
	})
})
