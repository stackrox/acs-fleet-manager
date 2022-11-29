package reconciler

import (
	corev1 "k8s.io/api/core/v1"
)

// TelemetryOptions defines parameters for pushing telemetry to a remote storage.
type TelemetryOptions struct {
	Endpoint   string
	StorageKey string
}

func getTelemetryEnvVars(telemetryOpts TelemetryOptions) []corev1.EnvVar {
	envVars := []corev1.EnvVar{
		{
			Name:  "ROX_TELEMETRY_ENDPOINT",
			Value: telemetryOpts.Endpoint,
		},
		{
			Name:  "ROX_TELEMETRY_STORAGE_KEY_V1",
			Value: telemetryOpts.StorageKey,
		},
	}
	return envVars
}
