package e2e

import (
	"context"

	. "github.com/onsi/gomega"

	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
)

func init() {
	apiTests["should have no traits"] = func(ctx context.Context, client *fleetmanager.Client, centralID string) {
		adminAPI := client.AdminAPI()
		traits, _, err := adminAPI.GetCentralTraits(ctx, centralID)
		Expect(err).ToNot(HaveOccurred())
		Expect(traits).To(BeEmpty())
		_, err = adminAPI.PatchCentralTraits(ctx, centralID, "test-trait")
		Expect(err).ToNot(HaveOccurred())
		traits, _, err = adminAPI.GetCentralTraits(ctx, centralID)
		Expect(err).ToNot(HaveOccurred())
		Expect(traits).To(BeEquivalentTo([]string{"test-trait"}))
		_, err = adminAPI.PatchCentralTraits(ctx, centralID, "test-trait-1")
		Expect(err).ToNot(HaveOccurred())
		_, err = adminAPI.PatchCentralTraits(ctx, centralID, "test-trait-1")
		Expect(err).ToNot(HaveOccurred())
		traits, _, err = adminAPI.GetCentralTraits(ctx, centralID)
		Expect(err).ToNot(HaveOccurred())
		Expect(traits).To(BeEquivalentTo([]string{"test-trait", "test-trait-1"}))

		_, err = adminAPI.PatchCentralTraits(ctx, centralID, "test-trait-2")
		Expect(err).To(HaveOccurred())
		_, err = adminAPI.DeleteCentralTrait(ctx, centralID, "test-trait")
		Expect(err).ToNot(HaveOccurred())
		_, err = adminAPI.DeleteCentralTrait(ctx, centralID, "test-trait")
		Expect(err).ToNot(HaveOccurred())
		traits, _, err = adminAPI.GetCentralTraits(ctx, centralID)
		Expect(err).ToNot(HaveOccurred())
		Expect(traits).To(BeEquivalentTo([]string{"test-trait-1"}))
	}
}
