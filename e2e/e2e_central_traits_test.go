package e2e

import (
	"context"

	. "github.com/onsi/gomega"

	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
)

func init() {
	apiTests["should manage traits"] = func(ctx context.Context, client *fleetmanager.Client, centralID string) {
		adminAPI := client.AdminAPI()
		traits, _, err := adminAPI.GetCentralTraits(ctx, centralID)
		Expect(err).ToNot(HaveOccurred(), "no error on no traits")
		Expect(traits).To(BeEmpty(), "no traits yet")

		_, err = adminAPI.PatchCentralTraits(ctx, centralID, "test-trait")
		Expect(err).ToNot(HaveOccurred(), "no error on adding test-trait")

		traits, _, err = adminAPI.GetCentralTraits(ctx, centralID)
		Expect(err).ToNot(HaveOccurred(), "no error on having traits")
		Expect(traits).To(BeEquivalentTo([]string{"test-trait"}), "test-trait should be found")

		_, err = adminAPI.PatchCentralTraits(ctx, centralID, "test-trait-1")
		Expect(err).ToNot(HaveOccurred(), "no error on adding test-trait-1")

		_, err = adminAPI.PatchCentralTraits(ctx, centralID, "test-trait-1")
		Expect(err).ToNot(HaveOccurred(), "no error on adding test-trait-1 twice")

		traits, _, err = adminAPI.GetCentralTraits(ctx, centralID)
		Expect(err).ToNot(HaveOccurred(), "no error on having multiple traits")
		Expect(traits).To(BeEquivalentTo([]string{"test-trait", "test-trait-1"}), "should have only two traits")

		_, err = adminAPI.GetCentralTrait(ctx, centralID, "test-trait")
		Expect(err).ToNot(HaveOccurred(), "no error on checking for existing trait")

		_, err = adminAPI.GetCentralTrait(ctx, centralID, "test-trait-2")
		Expect(err).To(HaveOccurred(), "error on checking for non-existing trait")

		_, err = adminAPI.DeleteCentralTrait(ctx, centralID, "test-trait")
		Expect(err).ToNot(HaveOccurred(), "no error on deleting test-trait")

		_, err = adminAPI.DeleteCentralTrait(ctx, centralID, "test-trait")
		Expect(err).ToNot(HaveOccurred(), "no error on deleting non-existing trait")

		traits, _, err = adminAPI.GetCentralTraits(ctx, centralID)
		Expect(err).ToNot(HaveOccurred(), "no error on retreiving traits")
		Expect(traits).To(BeEquivalentTo([]string{"test-trait-1"}), "should have only one trait now")
	}
}
