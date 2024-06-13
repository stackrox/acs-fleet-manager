package gitops

import (
	"github.com/prometheus/common/model"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"k8s.io/apimachinery/pkg/api/resource"
	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"time"
)

func validateVpaConfig(path *field.Path, vpaConfig *private.VerticalPodAutoscaling) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateVpaRecommenders(path.Child("recommenders"), vpaConfig.Recommenders)...)
	return allErrs
}

func validateVpaRecommenders(path *field.Path, recommenders []private.VpaRecommenderConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	seenNames := sets.NewString()
	for i, recommender := range recommenders {
		recommenderPath := path.Index(i)
		if seenNames.Has(recommender.Name) {
			allErrs = append(allErrs, field.Duplicate(recommenderPath.Child("name"), recommender.Name))
		}
		seenNames.Insert(recommender.Name)
		allErrs = append(allErrs, validateVpaRecommenderConfig(recommenderPath, &recommender)...)
	}
	return allErrs
}

func validateVpaRecommenderConfig(path *field.Path, recommender *private.VpaRecommenderConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	if recommender.Name == "" {
		allErrs = append(allErrs, field.Required(path.Child("name"), "name must be specified"))
	} else {
		if errs := apimachineryvalidation.NameIsDNSSubdomain(recommender.Name, false); len(errs) > 0 {
			allErrs = append(allErrs, field.Invalid(path.Child("name"), recommender.Name, "invalid name: "+errs[0]))
		}
	}
	if recommender.Image == "" {
		allErrs = append(allErrs, field.Required(path.Child("image"), "image must be specified"))
	}
	if recommender.RecommendationMarginFraction < 0 || recommender.RecommendationMarginFraction > 1 {
		allErrs = append(allErrs, field.Invalid(path.Child("recommendationMarginFraction"), recommender.RecommendationMarginFraction, "must be between 0 and 1"))
	}
	if recommender.PodRecommendationMinCpuMillicores < 0 {
		allErrs = append(allErrs, field.Invalid(path.Child("podRecommendationMinCpuMillicores"), recommender.PodRecommendationMinCpuMillicores, "must be non-negative"))
	}
	if recommender.PodRecommendationMinMemoryMb < 0 {
		allErrs = append(allErrs, field.Invalid(path.Child("podRecommendationMinMemoryMb"), recommender.PodRecommendationMinMemoryMb, "must be non-negative"))
	}
	if recommender.TargetMemoryPercentile < 0 || recommender.TargetMemoryPercentile > 1 {
		allErrs = append(allErrs, field.Invalid(path.Child("targetMemoryPercentile"), recommender.TargetMemoryPercentile, "must be between 0 and 1"))
	}
	if recommender.TargetCpuPercentile < 0 || recommender.TargetCpuPercentile > 1 {
		allErrs = append(allErrs, field.Invalid(path.Child("targetCpuPercentile"), recommender.TargetCpuPercentile, "must be between 0 and 1"))
	}
	if recommender.RecommendationLowerBoundMemoryPercentile < 0 || recommender.RecommendationLowerBoundMemoryPercentile > 1 {
		allErrs = append(allErrs, field.Invalid(path.Child("recommendationLowerBoundMemoryPercentile"), recommender.RecommendationLowerBoundMemoryPercentile, "must be between 0 and 1"))
	}
	if recommender.RecommendationUpperBoundMemoryPercentile < 0 || recommender.RecommendationUpperBoundMemoryPercentile > 1 {
		allErrs = append(allErrs, field.Invalid(path.Child("recommendationUpperBoundMemoryPercentile"), recommender.RecommendationUpperBoundMemoryPercentile, "must be between 0 and 1"))
	}
	if recommender.RecommendationLowerBoundCpuPercentile < 0 || recommender.RecommendationLowerBoundCpuPercentile > 1 {
		allErrs = append(allErrs, field.Invalid(path.Child("recommendationLowerBoundCpuPercentile"), recommender.RecommendationLowerBoundCpuPercentile, "must be between 0 and 1"))
	}
	if recommender.RecommendationUpperBoundCpuPercentile < 0 || recommender.RecommendationUpperBoundCpuPercentile > 1 {
		allErrs = append(allErrs, field.Invalid(path.Child("recommendationUpperBoundCpuPercentile"), recommender.RecommendationUpperBoundCpuPercentile, "must be between 0 and 1"))
	}
	if !isValidPromDuration(recommender.HistoryLength) {
		allErrs = append(allErrs, field.Invalid(path.Child("historyLength"), recommender.HistoryLength, "must be a valid duration"))
	}
	if !isValidPromDuration(recommender.HistoryResolution) {
		allErrs = append(allErrs, field.Invalid(path.Child("historyResolution"), recommender.HistoryResolution, "must be a valid duration"))
	}
	if !isValidDuration(recommender.RecommenderInterval) {
		allErrs = append(allErrs, field.Invalid(path.Child("recommenderInterval"), recommender.RecommenderInterval, "must be a valid duration"))
	}
	if !isValidDuration(recommender.CheckpointsTimeout) {
		allErrs = append(allErrs, field.Invalid(path.Child("checkpointsTimeout"), recommender.CheckpointsTimeout, "must be a valid duration"))
	}
	if !isValidDuration(recommender.CheckpointsGcInterval) {
		allErrs = append(allErrs, field.Invalid(path.Child("checkpointsGcInterval"), recommender.CheckpointsGcInterval, "must be a valid duration"))
	}
	if !isValidDuration(recommender.MemoryAggregationInterval) {
		allErrs = append(allErrs, field.Invalid(path.Child("memoryAggregationInterval"), recommender.MemoryAggregationInterval, "must be a valid duration"))
	}
	if !isValidDuration(recommender.MemoryHistogramDecayHalfLife) {
		allErrs = append(allErrs, field.Invalid(path.Child("memoryHistogramDecayHalfLife"), recommender.MemoryHistogramDecayHalfLife, "must be a valid duration"))
	}
	if !isValidDuration(recommender.CpuHistogramDecayHalfLife) {
		allErrs = append(allErrs, field.Invalid(path.Child("cpuHistogramDecayHalfLife"), recommender.CpuHistogramDecayHalfLife, "must be a valid duration"))
	}

	allErrs = append(allErrs, validateResourceRequirements(path.Child("resources"), recommender.Resources)...)

	return allErrs
}

func validateResourceRequirements(path *field.Path, r private.ResourceRequirements) (errs field.ErrorList) {
	if r.Requests != nil {
		errs = append(errs, validateResourceList(path.Child("requests"), r.Requests)...)
	}
	if r.Limits != nil {
		errs = append(errs, validateResourceList(path.Child("limits"), r.Limits)...)
	}
	return errs
}

func validateResourceList(path *field.Path, r map[string]string) (errs field.ErrorList) {
	for k, v := range r {
		if k != "cpu" && k != "memory" {
			errs = append(errs, field.NotSupported(path.Key(k), k, []string{"cpu", "memory"}))
			continue
		}
		_, err := resource.ParseQuantity(v)
		if err != nil {
			errs = append(errs, field.Invalid(path.Key(k), v, err.Error()))
		}
	}
	return errs
}

func isValidDuration(d string) bool {
	if len(d) == 0 {
		return true
	}
	_, err := time.ParseDuration(d)
	return err == nil
}

func isValidPromDuration(d string) bool {
	if len(d) == 0 {
		return true
	}
	_, err := model.ParseDuration(d)
	return err == nil
}
