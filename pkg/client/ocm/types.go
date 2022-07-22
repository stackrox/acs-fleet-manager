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

// RHOSAKProduct TODO change this to correspond to your own product types created in AMS
const (
	RHOSAKProduct      DinosaurProduct = "RHOSAK"      // this is the standard product type
	RHOSAKTrialProduct DinosaurProduct = "RHOSAKTrial" // this is trial product type which does not have any cost
)

// GetProduct ...
func (t DinosaurQuotaType) GetProduct() string {
	if t == StandardQuota {
		return string(RHOSAKProduct)
	}

	return string(RHOSAKTrialProduct)
}

// GetResourceName ...
func (t DinosaurQuotaType) GetResourceName() string {
	return "rhosak" // TODO change this to match your own AMS resource type. Usually it is the name of the product
}

// Equals ...
func (t DinosaurQuotaType) Equals(t1 DinosaurQuotaType) bool {
	return t1.GetProduct() == t.GetProduct() && t1.GetResourceName() == t.GetResourceName()
}
