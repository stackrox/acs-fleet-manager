package ocm

// Parameter ...
type Parameter struct {
	ID    string
	Value string
}

// CentralQuotaType ...
type CentralQuotaType string

// EvalQuota ...
const (
	EvalQuota     CentralQuotaType = "eval"
	StandardQuota CentralQuotaType = "standard"
)

// CentralProduct ...
type CentralProduct string

// RHACSProduct
const (
	RHACSProduct      CentralProduct = "RHACS"      // this is the standard product type
	RHACSTrialProduct CentralProduct = "RHACSTrial" // this is trial product type which does not have any cost
)

// GetProduct ...
func (t CentralQuotaType) GetProduct() string {
	if t == StandardQuota {
		return string(RHACSProduct)
	}

	return string(RHACSTrialProduct)
}

// GetResourceName ...
func (t CentralQuotaType) GetResourceName() string {
	return "rhacs"
}

// Equals ...
func (t CentralQuotaType) Equals(t1 CentralQuotaType) bool {
	return t1.GetProduct() == t.GetProduct() && t1.GetResourceName() == t.GetResourceName()
}
