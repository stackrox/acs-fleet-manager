package features

var (
	// TargetedOperatorUpgrades enables the targeted operator upgrades
	TargetedOperatorUpgrades = registerFeature("Upgrade Central instances targetedly via fleet-manager API", "RHACS_TARGETED_OPERATOR_UPGRADES", true)

	// GitOpsCentrals enables the GitOps for Central instances
	GitOpsCentrals = registerFeature("GitOps for Central instances", "RHACS_GITOPS_ENABLED", true)

	// PrintCentralUpdateDiff enables printing the diff of the central update
	PrintCentralUpdateDiff = registerFeature("Print the diff of the central update", "RHACS_PRINT_CENTRAL_UPDATE_DIFF", true)

	// PrintTenantResourcesChartValues enables printing the tenant resources chart values
	PrintTenantResourcesChartValues = registerFeature("Print the tenant resources chart values", "RHACS_PRINT_TENANT_RESOURCES_CHART_VALUES", false)

	// AddonAutoUpgrade enables addon auto upgrade feature
	AddonAutoUpgrade = registerFeature("Addon auto upgrade", "RHACS_ADDON_AUTO_UPGRADE", true)
)
