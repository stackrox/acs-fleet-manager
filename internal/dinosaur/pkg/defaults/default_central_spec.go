// Package defaults ...
package defaults

import (
	"fmt"

	"github.com/caarlos0/env/v6"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// CentralDefaults ...
type CentralDefaults struct {
	MemoryRequest resource.Quantity `env:"MEMORY_REQUEST" envDefault:"250Mi"`
	CPURequest    resource.Quantity `env:"CPU_REQUEST" envDefault:"50m"`
	MemoryLimit   resource.Quantity `env:"MEMORY_LIMIT" envDefault:"4G"`
	CPULimit      resource.Quantity `env:"CPU_LIMIT" envDefault:"250m"`
}

var (
	// Central ...
	Central CentralDefaults
	// CentralResources ...
	CentralResources corev1.ResourceRequirements
)

func init() {
	defaults := CentralDefaults{}
	opts := env.Options{
		Prefix: "CENTRAL_",
	}
	if err := env.ParseWithFuncs(&defaults, CustomParsers, opts); err != nil {
		panic(fmt.Sprintf("Unable to parse Central Defaults configuration from environment: %v", err))
	}
	Central = defaults
	CentralResources = corev1.ResourceRequirements{
		Requests: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    Central.CPURequest,
			corev1.ResourceMemory: Central.MemoryRequest,
		},
		Limits: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    Central.CPULimit,
			corev1.ResourceMemory: Central.MemoryLimit,
		},
	}
}
