package k8s

// ManagedByLabelKey ...
const (
	// ManagedByLabelKey indicates the tool being used to manage the operation of an application.
	ManagedByLabelKey = "app.kubernetes.io/managed-by"
	// ManagedByFleetshardValue used for indication that the resource is managed by the fleetshard sync
	ManagedByFleetshardValue = "rhacs-fleetshard"
)
