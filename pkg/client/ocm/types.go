package ocm

// Parameter ...
type Parameter struct {
	ID    string
	Value string
}

// DinosaurQuotaType ...
type DinosaurQuotaType string

// EvalQuota ...
const (
	EvalQuota     DinosaurQuotaType = "eval"
	StandardQuota DinosaurQuotaType = "standard"
)

// DinosaurProduct ...
type DinosaurProduct string

// RHACSProduct
const (
	RHACSProduct      DinosaurProduct = "RHACS"      // this is the standard product type
	RHACSTrialProduct DinosaurProduct = "RHACSTrial" // this is trial product type which does not have any cost
)

// GetProduct ...
func (t DinosaurQuotaType) GetProduct() string {
	if t == StandardQuota {
		return string(RHACSProduct)
	}

	return string(RHACSTrialProduct)
}

// GetResourceName ...
func (t DinosaurQuotaType) GetResourceName() string {
	return "rhacs"
}

// Equals ...
func (t DinosaurQuotaType) Equals(t1 DinosaurQuotaType) bool {
	return t1.GetProduct() == t.GetProduct() && t1.GetResourceName() == t.GetResourceName()
}
