package reconciler

const (
	endpointVariable   = "ROX_TELEMETRY_ENDPOINT"
	storageKeyVariable = "ROX_TELEMETRY_STORAGE_KEY_V1"
)

// TelemetryOptions defines parameters for pushing telemetry to a remote storage.
type TelemetryOptions struct {
	Endpoint   string
	StorageKey string
}
