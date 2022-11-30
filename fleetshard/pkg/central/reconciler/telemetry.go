package reconciler

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	endpointVariable   = "ROX_TELEMETRY_ENDPOINT"
	storageKeyVariable = "ROX_TELEMETRY_STORAGE_KEY_V1"
)

// TelemetryOptions defines parameters for pushing telemetry to a remote storage.
type TelemetryOptions struct {
	Endpoint   string
	StorageKey string
}

func appendNotEmpty(envVars []corev1.EnvVar, item corev1.EnvVar) []corev1.EnvVar {
	if item.Name != "" && item.Value != "" {
		envVars = append(envVars, item)
	}
	return envVars
}

func (t *TelemetryOptions) toEnvVars() []corev1.EnvVar {
	envVars := []corev1.EnvVar{}
	envVars = appendNotEmpty(envVars, corev1.EnvVar{
		Name:  endpointVariable,
		Value: t.Endpoint,
	})
	envVars = appendNotEmpty(envVars, corev1.EnvVar{
		Name:  storageKeyVariable,
		Value: t.StorageKey,
	})
	return envVars
}
