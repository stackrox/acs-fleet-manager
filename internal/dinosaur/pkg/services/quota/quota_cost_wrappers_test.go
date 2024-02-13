package quota

import (
	"testing"
	"time"

	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/stretchr/testify/assert"
)

func TestQuotaCostPrintWrapper(t *testing.T) {
	qc := makeStandardTestQuotaCost("resource", "orgID", 1, 2, t)
	w := (*quotaCostPrintWrapper)(qc)
	assert.Equal(t, `{orgID: "orgID", quotaID: "", allowed: 1, consumed: 2, accounts: [], resources: map[standard:[{provider: "", product: "RHACS"}]]}`,
		w.String())

	qc = makeCloudTestQuotaCost("resource", "orgID", 1, 2, t)
	w = (*quotaCostPrintWrapper)(qc)
	assert.Equal(t, `{orgID: "orgID", quotaID: "", allowed: 1, consumed: 2, accounts: [{accountID: "cloudAccountID", provider: "aws"}], resources: map[marketplace-aws:[{provider: "", product: "RHACS"}]]}`,
		w.String())

	qc1, _ := v1.NewQuotaCost().Allowed(1).OrganizationID("test-org").QuotaID("quota-id").RelatedResources(
		v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)),
	).Build()
	qc2, _ := v1.NewQuotaCost().Allowed(10).OrganizationID("test-org").RelatedResources(
		v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplaceAWS)),
	).CloudAccounts(
		v1.NewCloudAccount().CloudAccountID("test-account").CloudProviderID("test-provider"),
	).
		Build()
	mapped, _ := mapAllowedQuotaCosts([]*v1.QuotaCost{qc1, qc2})
	assert.Equal(t, `map[marketplace-aws:[{orgID: "test-org", quotaID: "", allowed: 10, consumed: 0, accounts: [{accountID: "test-account", provider: "test-provider"}], resources: map[marketplace-aws:[{provider: "", product: ""}]]}] standard:[{orgID: "test-org", quotaID: "quota-id", allowed: 1, consumed: 0, accounts: [], resources: map[standard:[{provider: "", product: ""}]]}]]`,
		printQuotaCostMap(mapped))

	ca, _ := v1.NewCloudAccount().CloudAccountID("id").CloudProviderID("provider").Contracts(
		v1.NewContract().StartDate(time.Time{}).EndDate(time.Time{}.Add(24 * time.Hour)),
	).Build()
	assert.Equal(t, `{accountID: "id", provider: "provider", contracts: [{0001-01-01 00:00:00 +0000 UTC -- 0001-01-02 00:00:00 +0000 UTC}]}`, (*cloudAccountWrapper)(ca).String())
}
