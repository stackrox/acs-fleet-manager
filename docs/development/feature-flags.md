# Feature Flags

Feature Flags can be added to the `pkg/features/list.go` file.
All feature flags must be prefixed with `RHACS_`.

Example in `list.go`: 
```
TargetedOperatorUpgrades = registerFeature("Upgrade Central instances targetedly via fleet-manager API", "RHACS_TARGETED_OPERATOR_UPGRADES", false)
```

The feature flag can be referenced in code globally:

```
if features.TargetedOperator.Upgrades.Enabled() {
  // do something...
}
```

`make deploy/dev` will automatically inject exported feature flags into your local deployments.
