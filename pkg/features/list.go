package features

var (
	// TargetedOperatorUpgrades enables the targeted operator upgrades
	TargetedOperatorUpgrades = registerFeature("Upgrade Central instances targetedly via fleet-manager API", "RHACS_TARGETED_OPERATOR_UPGRADES", false)

	// GitOpsCentrals enables the GitOps for Central instances
	GitOpsCentrals = registerFeature("GitOps for Central instances", "RHACS_GITOPS_ENABLED", true)

	// PrintCentralUpdateDiff enables printing the diff of the central update
	PrintCentralUpdateDiff = registerFeature("Print the diff of the central update", "RHACS_PRINT_CENTRAL_UPDATE_DIFF", true)

	// AddonAutoUpgrade enables addon auto upgrade feature
	AddonAutoUpgrade = registerFeature("Addon auto upgrade", "RHACS_ADDON_AUTO_UPGRADE", true)

	// LeaderElectionEnabled enables the leader election
	LeaderElectionEnabled = registerFeature("Leader election", "LEADER_ELECTION_ENABLED", true)
)
