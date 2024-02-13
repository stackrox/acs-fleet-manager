package quota

import (
	"fmt"

	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

type relatedResourceStringWrapper amsv1.RelatedResource

func (r *relatedResourceStringWrapper) String() string {
	rr := (*amsv1.RelatedResource)(r)
	return fmt.Sprintf("{provider: %q, product: %q}", rr.CloudProvider(), rr.Product())
}

type cloudAccountStringWrapper amsv1.CloudAccount

func (a *cloudAccountStringWrapper) String() string {
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

type quotaCostStringWrapper amsv1.QuotaCost

func (q *quotaCostStringWrapper) String() string {
	qc := (*amsv1.QuotaCost)(q)
	cas := make([]*cloudAccountStringWrapper, 0, len(qc.CloudAccounts()))
	for _, ca := range qc.CloudAccounts() {
		cas = append(cas, (*cloudAccountStringWrapper)(ca))
	}
	res := map[string][]string{}
	for _, r := range qc.RelatedResources() {
		res[r.BillingModel()] = append(res[r.BillingModel()], (*relatedResourceStringWrapper)(r).String())
	}
	return fmt.Sprintf("{orgID: %q, quotaID: %q, allowed: %d, consumed: %d, accounts: %v, resources: %v}",
		qc.OrganizationID(), qc.QuotaID(), qc.Allowed(), qc.Consumed(), cas, res)
}

func printQuotaCostMap(m map[amsv1.BillingModel][]*amsv1.QuotaCost) string {
	wrappedQuotaCost := make(map[amsv1.BillingModel][]*quotaCostStringWrapper, len(m))
	for model, qcs := range m {
		for _, qc := range qcs {
			wrappedQuotaCost[model] = append(wrappedQuotaCost[model], (*quotaCostStringWrapper)(qc))
		}
	}
	return fmt.Sprint(wrappedQuotaCost)
}
