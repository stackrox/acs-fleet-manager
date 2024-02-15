package e2e

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	fmImpl "github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager/impl"
)

var _ = Describe("central traits", Ordered, func() {

	var client *fleetmanager.Client
	var adminAPI fleetmanager.AdminAPI
	var notes []string
	var ctx = context.Background()

	BeforeEach(func() {
		SkipIf(!runCentralTests, "Skipping Central tests")
		Expect(restoreDefaultGitopsConfig()).To(Succeed())

		option := fmImpl.OptionFromEnv()
		auth, err := fmImpl.NewStaticAuth(context.Background(), fmImpl.StaticOption{StaticToken: option.Static.StaticToken})
		Expect(err).ToNot(HaveOccurred())
		client, err = fmImpl.NewClient(fleetManagerEndpoint, auth)
		Expect(err).ToNot(HaveOccurred())

		adminStaticToken := os.Getenv("STATIC_TOKEN_ADMIN")
		adminAuth, err := fmImpl.NewStaticAuth(context.Background(), fmImpl.StaticOption{StaticToken: adminStaticToken})
		Expect(err).ToNot(HaveOccurred())
		adminClient, err := fmImpl.NewClient(fleetManagerEndpoint, adminAuth)
		Expect(err).ToNot(HaveOccurred())
		adminAPI = adminClient.AdminAPI()

		GinkgoWriter.Printf("Current time: %s\n", time.Now().String())
		printNotes(notes)
	})

	It("should manage traits", func() {
		central, _, err := adminAPI.CreateCentral(ctx, true, private.CentralRequestPayload{
			CloudProvider: dpCloudProvider,
			MultiAz:       true,
			Name:          newCentralName(),
			Region:        dpRegion,
		})
		Expect(err).Should(Succeed())
		id := central.Id

		Eventually(assertCentralRequestProvisioning(ctx, client, id)).
			WithTimeout(waitTimeout).
			WithPolling(defaultPolling).
			Should(Succeed())

		defer adminAPI.DeleteDbCentralById(ctx, id)

		traits, _, err := adminAPI.GetCentralTraits(ctx, id)
		Expect(err).ToNot(HaveOccurred(), "no error on no traits")
		Expect(traits).To(BeEmpty(), "no traits yet")

		_, err = adminAPI.PutCentralTrait(ctx, id, "test-trait")
		Expect(err).ToNot(HaveOccurred(), "no error on adding test-trait")

		traits, _, err = adminAPI.GetCentralTraits(ctx, id)
		Expect(err).ToNot(HaveOccurred(), "no error on having traits")
		Expect(traits).To(BeEquivalentTo([]string{"test-trait"}), "test-trait should be found")

		_, err = adminAPI.PutCentralTrait(ctx, id, "test-trait-1")
		Expect(err).ToNot(HaveOccurred(), "no error on adding test-trait-1")

		_, err = adminAPI.PutCentralTrait(ctx, id, "test-trait-1")
		Expect(err).ToNot(HaveOccurred(), "no error on adding test-trait-1 twice")

		traits, _, err = adminAPI.GetCentralTraits(ctx, id)
		Expect(err).ToNot(HaveOccurred(), "no error on having multiple traits")
		Expect(traits).To(BeEquivalentTo([]string{"test-trait", "test-trait-1"}), "should have only two traits")

		_, err = adminAPI.GetCentralTrait(ctx, id, "test-trait")
		Expect(err).ToNot(HaveOccurred(), "no error on checking for existing trait")

		_, err = adminAPI.GetCentralTrait(ctx, id, "test-trait-2")
		Expect(err).To(HaveOccurred(), "error on checking for non-existing trait")

		_, err = adminAPI.DeleteCentralTrait(ctx, id, "test-trait")
		Expect(err).ToNot(HaveOccurred(), "no error on deleting test-trait")

		_, err = adminAPI.DeleteCentralTrait(ctx, id, "test-trait")
		Expect(err).ToNot(HaveOccurred(), "no error on deleting non-existing trait")

		traits, _, err = adminAPI.GetCentralTraits(ctx, id)
		Expect(err).ToNot(HaveOccurred(), "no error on retreiving traits")
		Expect(traits).To(BeEquivalentTo([]string{"test-trait-1"}), "should have only one trait now")
	})
})
