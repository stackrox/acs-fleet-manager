package reconciler

// AuthInitStatus represents state machine for auth initialisation in Managed Central.
// There are two possible execution flows for this state machine:
// CreateRhSso -> DisableAdminPassword -> AuthInitCompleted
// or
// AuthInitCompleted
type AuthInitStatus int

const (
	// CreateRhSso signals reconciler to create sso.redhat.com auth provider.
	CreateRhSso AuthInitStatus = iota
	// DisableAdminPassword signals reconciler to disable admin password.
	DisableAdminPassword
	// AuthInitCompleted signals that auth initialisation is completed.
	AuthInitCompleted
)
