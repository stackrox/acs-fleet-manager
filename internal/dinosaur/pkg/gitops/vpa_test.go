package gitops

import (
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"testing"
)

func TestValidateRecommenders_duplicateNames(t *testing.T) {

	recommenders := []private.VpaRecommenderConfig{
		{
			Name:  "foo",
			Image: "bla",
		},
		{
			Name:  "foo",
			Image: "bla",
		},
	}

	errs := validateVpaRecommenders(field.NewPath("recommenders"), recommenders)

	require.Len(t, errs, 1)
	assert.Equal(t, errs[0].Type, field.ErrorTypeDuplicate)

}

func TestValidateRecommender(t *testing.T) {

	noError := func(t *testing.T, errs field.ErrorList) {
		assert.Empty(t, errs)
	}

	hasError := func(f, message string) func(t *testing.T, errs field.ErrorList) {
		return func(t *testing.T, errs field.ErrorList) {
			require.Len(t, errs, 1)
			assert.Equal(t, f, errs[0].Field)
			assert.Contains(t, errs[0].Detail, message)
		}
	}

	tests := []struct {
		name        string
		recommender private.VpaRecommenderConfig
		assert      func(t *testing.T, errs field.ErrorList)
	}{
		{
			name: "minimal",
			recommender: private.VpaRecommenderConfig{
				Name:  "foo",
				Image: "bla",
			},
			assert: noError,
		},
		{
			name: "full",
			recommender: private.VpaRecommenderConfig{
				Name:  "vpa",
				Image: "image",
				ImagePullSecrets: []private.LocalObjectReference{
					{
						Name: "secret",
					},
				},
				Resources: private.ResourceRequirements{
					Requests: map[string]string{
						"cpu":    "100m",
						"memory": "100Mi",
					},
					Limits: map[string]string{
						"cpu":    "100m",
						"memory": "100Mi",
					},
				},
				RecommendationMarginFraction:             0.1,
				PodRecommendationMinCpuMillicores:        100,
				PodRecommendationMinMemoryMb:             0.1,
				TargetCpuPercentile:                      0.1,
				RecommendationLowerBoundCpuPercentile:    0.1,
				RecommendationUpperBoundCpuPercentile:    0.1,
				TargetMemoryPercentile:                   0.1,
				RecommendationLowerBoundMemoryPercentile: 0.1,
				RecommendationUpperBoundMemoryPercentile: 0.1,
				CheckpointsTimeout:                       "1h",
				MinCheckpoints:                           10,
				MemorySaver:                              true,
				RecommenderInterval:                      "1h",
				CheckpointsGcInterval:                    "1h",
				PrometheusAddress:                        "address",
				PrometheusCadvisorJobName:                "job",
				Address:                                  "address",
				Kubeconfig:                               "abc",
				KubeApiQps:                               10,
				KubeApiBurst:                             10,
				Storage:                                  "storage",
				HistoryLength:                            "1h",
				HistoryResolution:                        "1h",
				PrometheusQueryTimeout:                   "1h",
				PodLabelPrefix:                           "prefix",
				MetricForPodLabels:                       "metric",
				PodNamespaceLabel:                        "label",
				PodNameLabel:                             "label",
				ContainerNamespaceLabel:                  "label",
				ContainerPodNameLabel:                    "label",
				ContainerNameLabel:                       "label",
				VpaObjectNamespace:                       "namespace",
				MemoryAggregationInterval:                "1h",
				MemoryAggregationIntervalCount:           10,
				MemoryHistogramDecayHalfLife:             "1h",
				CpuHistogramDecayHalfLife:                "1h",
				CpuIntegerPostProcessorEnabled:           true,
			},
			assert: noError,
		},
		{
			name: "recommendationMarginFraction",
			recommender: private.VpaRecommenderConfig{
				Name:                         "foo",
				Image:                        "bla",
				RecommendationMarginFraction: 1.1,
			},
			assert: hasError("recommender.recommendationMarginFraction", "must be between 0 and 1"),
		},
		{
			name: "podRecommendationMinCpuMillicores",
			recommender: private.VpaRecommenderConfig{
				Name:                              "foo",
				Image:                             "bla",
				PodRecommendationMinCpuMillicores: -1,
			},
			assert: hasError("recommender.podRecommendationMinCpuMillicores", "must be non-negative"),
		},
		{
			name: "podRecommendationMinMemoryMb",
			recommender: private.VpaRecommenderConfig{
				Name:                         "foo",
				Image:                        "bla",
				PodRecommendationMinMemoryMb: -1,
			},
			assert: hasError("recommender.podRecommendationMinMemoryMb", "must be non-negative"),
		},
		{
			name: "targetCpuPercentile",
			recommender: private.VpaRecommenderConfig{
				Name:                "foo",
				Image:               "bla",
				TargetCpuPercentile: 1.1,
			},
			assert: hasError("recommender.targetCpuPercentile", "must be between 0 and 1"),
		},
		{
			name: "targetMemoryPercentile",
			recommender: private.VpaRecommenderConfig{
				Name:                   "foo",
				Image:                  "bla",
				TargetMemoryPercentile: 1.1,
			},
			assert: hasError("recommender.targetMemoryPercentile", "must be between 0 and 1"),
		},
		{
			name: "recommendationLowerBoundMemoryPercentile",
			recommender: private.VpaRecommenderConfig{
				Name:                                     "foo",
				Image:                                    "bla",
				RecommendationLowerBoundMemoryPercentile: 1.1,
			},
			assert: hasError("recommender.recommendationLowerBoundMemoryPercentile", "must be between 0 and 1"),
		},
		{
			name: "recommendationUpperBoundMemoryPercentile",
			recommender: private.VpaRecommenderConfig{
				Name:                                     "foo",
				Image:                                    "bla",
				RecommendationUpperBoundMemoryPercentile: 1.1,
			},
			assert: hasError("recommender.recommendationUpperBoundMemoryPercentile", "must be between 0 and 1"),
		},
		{
			name: "recommendationLowerBoundCpuPercentile",
			recommender: private.VpaRecommenderConfig{
				Name:                                  "foo",
				Image:                                 "bla",
				RecommendationLowerBoundCpuPercentile: 1.1,
			},
			assert: hasError("recommender.recommendationLowerBoundCpuPercentile", "must be between 0 and 1"),
		},
		{
			name: "recommendationUpperBoundCpuPercentile",
			recommender: private.VpaRecommenderConfig{
				Name:                                  "foo",
				Image:                                 "bla",
				RecommendationUpperBoundCpuPercentile: 1.1,
			},
			assert: hasError("recommender.recommendationUpperBoundCpuPercentile", "must be between 0 and 1"),
		},
		{
			name: "missingName",
			recommender: private.VpaRecommenderConfig{
				Image: "bla",
			},
			assert: hasError("recommender.name", "name must be specified"),
		},
		{
			name: "missingImage",
			recommender: private.VpaRecommenderConfig{
				Name: "bla",
			},
			assert: hasError("recommender.image", "image must be specified"),
		},
		{
			name: "historyLength",
			recommender: private.VpaRecommenderConfig{
				Name:          "foo",
				Image:         "bla",
				HistoryLength: "1",
			},
			assert: hasError("recommender.historyLength", "must be a valid duration"),
		},
		{
			name: "historyResolution",
			recommender: private.VpaRecommenderConfig{
				Name:              "foo",
				Image:             "bla",
				HistoryResolution: "1",
			},
			assert: hasError("recommender.historyResolution", "must be a valid duration"),
		},
		{
			name: "recommenderInterval",
			recommender: private.VpaRecommenderConfig{
				Name:                "foo",
				Image:               "bla",
				RecommenderInterval: "1",
			},
			assert: hasError("recommender.recommenderInterval", "must be a valid duration"),
		},
		{
			name: "checkpointsTimeout",
			recommender: private.VpaRecommenderConfig{
				Name:               "foo",
				Image:              "bla",
				CheckpointsTimeout: "1",
			},
			assert: hasError("recommender.checkpointsTimeout", "must be a valid duration"),
		},
		{
			name: "checkpointsGcInterval",
			recommender: private.VpaRecommenderConfig{
				Name:                  "foo",
				Image:                 "bla",
				CheckpointsGcInterval: "1",
			},
			assert: hasError("recommender.checkpointsGcInterval", "must be a valid duration"),
		},
		{
			name: "memoryAggregationInterval",
			recommender: private.VpaRecommenderConfig{
				Name:                      "foo",
				Image:                     "bla",
				MemoryAggregationInterval: "1",
			},
			assert: hasError("recommender.memoryAggregationInterval", "must be a valid duration"),
		},
		{
			name: "memoryHistogramDecayHalfLife",
			recommender: private.VpaRecommenderConfig{
				Name:                         "foo",
				Image:                        "bla",
				MemoryHistogramDecayHalfLife: "1",
			},
			assert: hasError("recommender.memoryHistogramDecayHalfLife", "must be a valid duration"),
		},
		{
			name: "cpuHistogramDecayHalfLife",
			recommender: private.VpaRecommenderConfig{
				Name:                      "foo",
				Image:                     "bla",
				CpuHistogramDecayHalfLife: "1",
			},
			assert: hasError("recommender.cpuHistogramDecayHalfLife", "must be a valid duration"),
		},
		{
			name: "bad name",
			recommender: private.VpaRecommenderConfig{
				Name:  "foo/bar",
				Image: "bla",
			},
			assert: hasError("recommender.name", "invalid name"),
		},
		{
			name: "bad resources",
			recommender: private.VpaRecommenderConfig{
				Name:  "foo",
				Image: "bla",
				Resources: private.ResourceRequirements{
					Requests: map[string]string{
						"cpu": "100m",
						"bla": "100Mi",
					},
				},
			},
			assert: hasError("recommender.resources.requests[bla]", `supported values: "cpu", "memory"`),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			errs := validateVpaRecommenderConfig(field.NewPath("recommender"), &tt.recommender)
			tt.assert(t, errs)
		})
	}

}
