package constants

const (
	// OperationCreate is a general name for create operation for centrals/clusters.
	OperationCreate = "create"

	// OperationDelete is a general name for delete operation for centrals/clusters.
	OperationDelete = "delete"
)

// Operations is a list of possible operations values.
// It is used to initialize corresponding metrics with zero value.
var Operations = []string{OperationCreate, OperationDelete}
