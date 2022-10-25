package defaults

import (
	"fmt"

	"github.com/caarlos0/env/v6"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// AnalyzerDefaults ...
type AnalyzerDefaults struct {
	MemoryRequest resource.Quantity `env:"MEMORY_REQUEST" envDefault:"100Mi"`
	CPURequest    resource.Quantity `env:"CPU_REQUEST"    envDefault:"10m"`
	MemoryLimit   resource.Quantity `env:"MEMORY_LIMIT"   envDefault:"2500Mi"`
	CPULimit      resource.Quantity `env:"CPU_LIMIT"      envDefault:"250m"`
	AutoScaling   string            `env:"AUTOSCALING"    envDefault:"Enabled"`
	MinReplicas   int32             `env:"MIN_REPLICAS"   envDefault:"1"`
	Replicas      int32             `env:"REPLICAS"       envDefault:"1"`
	MaxReplicas   int32             `env:"MAX_REPLICAS"   envDefault:"3"`
}

// DbDefaults ...
type DbDefaults struct {
	MemoryRequest resource.Quantity `env:"MEMORY_REQUEST" envDefault:"500Mi"`
	CPURequest    resource.Quantity `env:"CPU_REQUEST"    envDefault:"10m"`
	MemoryLimit   resource.Quantity `env:"MEMORY_LIMIT"   envDefault:"2500Mi"`
	CPULimit      resource.Quantity `env:"CPU_LIMIT"      envDefault:"250m"`
}

// ScannerDefaults ...
type ScannerDefaults struct {
	Analyzer AnalyzerDefaults `envPrefix:"ANALYZER_"`
	Db       DbDefaults       `envPrefix:"DB_"`
}

var (
	// Scanner ...
	Scanner ScannerDefaults
	// ScannerAnalyzerResources ...
	ScannerAnalyzerResources corev1.ResourceRequirements
	// ScannerDbResources ...
	ScannerDbResources corev1.ResourceRequirements
)

func init() {
	defaults := ScannerDefaults{}
	opts := env.Options{
		Prefix: "SCANNER_",
	}
	if err := env.ParseWithFuncs(&defaults, CustomParsers, opts); err != nil {
		panic(fmt.Sprintf("Unable to parse Central Defaults configuration from environment: %v", err))
	}
	Scanner = defaults
	ScannerAnalyzerResources = corev1.ResourceRequirements{
		Requests: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    Scanner.Analyzer.CPURequest,
			corev1.ResourceMemory: Scanner.Analyzer.MemoryRequest,
		},
		Limits: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    Scanner.Analyzer.CPULimit,
			corev1.ResourceMemory: Scanner.Analyzer.MemoryLimit,
		},
	}
	ScannerDbResources = corev1.ResourceRequirements{
		Requests: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    Scanner.Db.CPURequest,
			corev1.ResourceMemory: Scanner.Db.MemoryRequest,
		},
		Limits: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    Scanner.Db.CPULimit,
			corev1.ResourceMemory: Scanner.Db.MemoryLimit,
		},
	}
}
