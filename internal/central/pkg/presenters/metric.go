package presenters

import (
	pmod "github.com/prometheus/common/model"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/pkg/client/observatorium"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

func convertMatrix(from pmod.Matrix) []public.RangeQuery {
	series := make([]public.RangeQuery, len(from))

	for i, s := range from {
		series[i] = convertSampleStream(s)
	}
	return series
}
func convertVector(from pmod.Vector) []public.InstantQuery {
	series := make([]public.InstantQuery, len(from))

	for i, s := range from {
		series[i] = convertSample(s)
	}
	return series
}

func convertSampleStream(from *pmod.SampleStream) public.RangeQuery {
	labelSet := make(map[string]string, len(from.Metric))
	for k, v := range from.Metric {
		if !isAllowedLabel(string(k)) {
			// Do not add these labels
			continue
		}
		labelSet[string(k)] = string(v)
	}
	values := make([]public.Values, len(from.Values))
	for i := range from.Values {
		values[i] = convertSamplePair(&from.Values[i])
	}
	return public.RangeQuery{
		Metric: labelSet,
		Values: values,
	}
}
func convertSample(from *pmod.Sample) public.InstantQuery {
	labelSet := make(map[string]string, len(from.Metric))
	for k, v := range from.Metric {
		if !isAllowedLabel(string(k)) {
			// Do not add these labels
			continue
		}
		labelSet[string(k)] = string(v)
	}
	return public.InstantQuery{
		Metric:    labelSet,
		Timestamp: int64(from.Timestamp),
		Value:     float64(from.Value),
	}
}

func convertSamplePair(from *pmod.SamplePair) public.Values {
	return public.Values{
		Timestamp: int64(from.Timestamp),
		Value:     float64(from.Value),
	}
}

// PresentMetricsByRangeQuery ...
func PresentMetricsByRangeQuery(metrics *observatorium.DinosaurMetrics) ([]public.RangeQuery, *errors.ServiceError) {
	out := []public.RangeQuery{}
	for _, m := range *metrics {
		if m.Err != nil {
			return nil, errors.GeneralError("error in metric %s: %v", m.Matrix, m.Err)
		}
		metric := convertMatrix(m.Matrix)
		out = append(out, metric...)
	}
	return out, nil
}

// PresentMetricsByInstantQuery ...
func PresentMetricsByInstantQuery(metrics *observatorium.DinosaurMetrics) ([]public.InstantQuery, *errors.ServiceError) {
	out := []public.InstantQuery{}
	for _, m := range *metrics {
		if m.Err != nil {
			return nil, errors.GeneralError("error in metric %s: %v", m.Matrix, m.Err)
		}
		metric := convertVector(m.Vector)
		out = append(out, metric...)
	}
	return out, nil
}

func isAllowedLabel(lable string) bool {
	for _, labelName := range getSupportedLables() {
		if lable == labelName {
			return true
		}
	}

	return false
}

// TODO change supported labels to match the metrics labels supported for your service
func getSupportedLables() []string {
	return []string{"__name__", "dinosaur_operator_io_cluster", "topic", "persistentvolumeclaim", "statefulset_kubernetes_io_pod_name", "exported_service", "exported_pod", "route"}
}
