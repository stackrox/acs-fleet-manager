package features

var (
	// TargetedOperatorUpgrades enables the targeted operator upgrades
	TargetedOperatorUpgrades = registerFeature("Upgrade Central instances targetedly via fleet-manager API", "RHACS_TARGETED_OPERATOR_UPGRADES", false)

	// PrintCentralUpdateDiff enables printing the diff of the central update
	PrintCentralUpdateDiff = registerFeature("Print the diff of the central update", "RHACS_PRINT_CENTRAL_UPDATE_DIFF", false)

	// StandaloneMode makes Fleetshard-sync service use ACS Operators configuration from ConfigMap
	StandaloneMode = registerFeature("Use ACS Operators configuration from ConfigMap", "RHACS_STANDALONE_MODE", false)
)
