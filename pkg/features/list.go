package features

var (
	// TargetedOperatorUpgrades enables the targeted operator upgrades
	TargetedOperatorUpgrades = registerFeature("Upgrade Central instances targetedly via fleet-manager API", "RHACS_TARGETED_OPERATOR_UPGRADES", false)

	// GitOpsCentrals enables the GitOps for Central instances
	GitOpsCentrals = registerFeature("GitOps for Central instances", "RHACS_GITOPS_ENABLED", false)

	// PrintCentralUpdateDiff enables printing the diff of the central update
	PrintCentralUpdateDiff = registerFeature("Print the diff of the central update", "RHACS_PRINT_CENTRAL_UPDATE_DIFF", false)

	// StandaloneMode makes Fleetshard-sync service use ACS Operators configuration from ConfigMap
	StandaloneMode = registerFeature("Use ACS Operators configuration from ConfigMap", "RHACS_STANDALONE_MODE", false)
)
