package features

var (
	// TargetedOperatorUpgrades enables the targeted operator upgrades
	TargetedOperatorUpgrades = registerFeature("Upgrade Central instances targetedly via fleet-manager API", "RHACS_TARGETED_OPERATOR_UPGRADES", true)

	// GitOpsCentrals enables the GitOps for Central instances
	GitOpsCentrals = registerFeature("GitOps for Central instances", "RHACS_GITOPS_ENABLED", true)

	// AddonAutoUpgrade enables addon auto upgrade feature
	AddonAutoUpgrade = registerFeature("Addon auto upgrade", "RHACS_ADDON_AUTO_UPGRADE", true)

	// ClusterMigration enables the feature to migrate a tenant to another cluster
	ClusterMigration = registerFeature("Cluster migraiton", "RHACS_CLUSTER_MIGRATION", true)
)
