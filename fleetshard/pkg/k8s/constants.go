package k8s

// ManagedByLabelKey ...
const (
	// ManagedByLabelKey indicates the tool being used to manage the operation of an application.
	ManagedByLabelKey = "app.kubernetes.io/managed-by"
	// ManagedByFleetshardValue used for indication that the resource is managed by the fleetshard sync
	ManagedByFleetshardValue = "rhacs-fleetshard"
	// VersionSelectorLabelKey is the label the rhacs-operator uses (via its CENTRAL_LABEL_SELECTOR
	// setting) to decide which Central CRs it reconciles. It is stamped onto the Central CR by the
	// tenant-resources Helm chart from the tenant's rolloutGroup value, allowing multiple operator
	// versions to coexist during a canary upgrade.
	VersionSelectorLabelKey = "rhacs.redhat.com/version-selector"
)
