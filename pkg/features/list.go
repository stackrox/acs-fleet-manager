package features

var (
	// TargetedOperatorUpgrades enables the targeted operator upgrades
	TargetedOperatorUpgrades = registerFeature("Upgrade Central instances targetedly via fleet-manager API", "RHACS_TARGETED_OPERATOR_UPGRADES", false)

	// PrintCentralUpdateDiff enables printing the diff of the central update
	PrintCentralUpdateDiff = registerFeature("Print the diff of the central update", "RHACS_PRINT_CENTRAL_UPDATE_DIFF", false)

	// UseOperatorsConfigMap makes Fleetshard-sync service use ACS Operators configuration from ConfigMap. It is useful for E2E testing
	UseOperatorsConfigMap = registerFeature("Use ACS Operators configuration from ConfigMap", "RHACS_DEBUG_MODE", false)
)
