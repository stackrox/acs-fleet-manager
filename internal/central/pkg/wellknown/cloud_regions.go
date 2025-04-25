package wellknown

// cloudRegionDisplayNames is a map of cloud provider names to a map of region names to display names.
// Obtained from https://docs.aws.amazon.com/general/latest/gr/rande.html
var cloudRegionDisplayNames = map[string]map[string]string{
	"aws": {
		"af-south-1":     "Africa (Cape Town)",
		"ap-east-1":      "Asia Pacific (Hong Kong)",
		"ap-northeast-1": "Asia Pacific (Tokyo)",
		"ap-northeast-2": "Asia Pacific (Seoul)",
		"ap-northeast-3": "Asia Pacific (Osaka)",
		"ap-south-1":     "Asia Pacific (Mumbai)",
		"ap-south-2":     "Asia Pacific (Hyderabad)",
		"ap-southeast-1": "Asia Pacific (Singapore)",
		"ap-southeast-2": "Asia Pacific (Sydney)",
		"ap-southeast-3": "Asia Pacific (Jakarta)",
		"ap-southeast-4": "Asia Pacific (Melbourne)",
		"ca-central-1":   "Canada (Central)",
		"eu-central-1":   "Europe (Frankfurt)",
		"eu-central-2":   "Europe (Zurich)",
		"eu-north-1":     "Europe (Stockholm)",
		"eu-south-1":     "Europe (Milan)",
		"eu-south-2":     "Europe (Spain)",
		"eu-west-1":      "Europe (Ireland)",
		"eu-west-2":      "Europe (London)",
		"eu-west-3":      "Europe (Paris)",
		"me-central-1":   "Middle East (UAE)",
		"me-south-1":     "Middle East (Bahrain)",
		"sa-east-1":      "South America (São Paulo)",
		"us-east-1":      "US East (N. Virginia)",
		"us-east-2":      "US East (Ohio)",
		"us-gov-east-1":  "AWS GovCloud (US-East)",
		"us-gov-west-1":  "AWS GovCloud (US-West)",
		"us-west-1":      "US West (N. California)",
		"us-west-2":      "US West (Oregon)",
	},
}

// GetCloudRegionDisplayName returns the display name for a cloud region.
// If the provider or region is unknown, the input region name is returned.
func GetCloudRegionDisplayName(providerName, regionName string) string {
	if regions, ok := cloudRegionDisplayNames[providerName]; ok {
		if displayName, ok := regions[regionName]; ok {
			return displayName
		}
	}
	return regionName
}
