package quota

import (
	"fmt"

	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

type relatedResourcePrintWrapper amsv1.RelatedResource

func (r *relatedResourcePrintWrapper) String() string {
	rr := (*amsv1.RelatedResource)(r)
	return fmt.Sprintf("{provider: %q, product: %q}", rr.CloudProvider(), rr.Product())
}

type cloudAccountWrapper amsv1.CloudAccount

func (a *cloudAccountWrapper) String() string {
	ca := (*amsv1.CloudAccount)(a)
	if len(ca.Contracts()) == 0 {
		return fmt.Sprintf("{accountID: %q, provider: %q}", ca.CloudAccountID(), ca.CloudProviderID())
	}
	contractsList := make([]string, 0, len(ca.Contracts()))
	for _, contract := range ca.Contracts() {
		contractsList = append(contractsList, fmt.Sprintf("{%v -- %v}", contract.StartDate(), contract.EndDate()))
	}
	return fmt.Sprintf("{accountID: %q, provider: %q, contracts: %v}", ca.CloudAccountID(), ca.CloudProviderID(), contractsList)
}

type quotaCostPrintWrapper amsv1.QuotaCost

func (q *quotaCostPrintWrapper) String() string {
	qc := (*amsv1.QuotaCost)(q)
	cas := make([]*cloudAccountWrapper, 0, len(qc.CloudAccounts()))
	for _, ca := range qc.CloudAccounts() {
		cas = append(cas, (*cloudAccountWrapper)(ca))
	}
	res := map[string][]string{}
	for _, r := range qc.RelatedResources() {
		res[r.BillingModel()] = append(res[r.BillingModel()], (*relatedResourcePrintWrapper)(r).String())
	}
	return fmt.Sprintf("{orgID: %q, quotaID: %q, allowed: %d, consumed: %d, accounts: %v, resources: %v}",
		qc.OrganizationID(), qc.QuotaID(), qc.Allowed(), qc.Consumed(), cas, res)
}

func printQuotaCostMap(m map[amsv1.BillingModel][]*amsv1.QuotaCost) string {
	wrappedQuotaCost := make(map[amsv1.BillingModel][]*quotaCostPrintWrapper, len(m))
	for model, qcs := range m {
		for _, qc := range qcs {
			wrappedQuotaCost[model] = append(wrappedQuotaCost[model], (*quotaCostPrintWrapper)(qc))
		}
	}
	return fmt.Sprint(wrappedQuotaCost)
}
