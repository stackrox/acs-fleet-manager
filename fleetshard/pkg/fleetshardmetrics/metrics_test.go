package fleetshardmetrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
)

func TestMetricIncrements(t *testing.T) {
	tt := []struct {
		metricName    string
		incrementFunc func()
	}{
		{
			metricName:    "total_k8s_requests",
			incrementFunc: IncrementK8sRequests,
		},
		{
			metricName:    "total_k8s_request_errors",
			incrementFunc: IncrementK8sRequestErrors,
		},
		{
			metricName:    "total_fleet_manager_requests",
			incrementFunc: IncrementFleetManagerRequests,
		},
		{
			metricName:    "total_fleet_manager_request_errors",
			incrementFunc: IncrementFleetManagerRequestErrors,
		},
	}
	promParser := expfmt.TextParser{}

	for _, tc := range tt {
		t.Run(tc.metricName, func(t *testing.T) {
			resetMetrics()
			handler := NewMetricsServer(":8081").Handler
			req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
			assert.NoError(t, err, "failed creating metrics requests")

			// Call the increment function
			tc.incrementFunc()

			// Test that the metrics value is 1 after calling the incrementFunc
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code, "status code should be OK")

			metrics, err := promParser.TextToMetricFamilies(rec.Body)
			assert.NoError(t, err, "failed creating metrics requests")
			targetMetric, hasKey := metrics[tc.metricName]
			assert.Truef(t, hasKey, "expected metrics to contain %s but it did not: %v", tc.metricName, metrics)

			value := targetMetric.Metric[0].Counter.Value
			assert.Equalf(t, 1.0, *value, "expected metric: %s to have value: %v", tc.metricName, value)
		})
	}
}

func resetMetrics() {
	k8sRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "total_k8s_requests",
		Help: "The total number of requests send to the target kubernetes cluster",
	})

	k8sRequestErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "total_k8s_request_errors",
		Help: "The total number of unexpected errors for requests send to the target kubernetes cluster",
	})

	fleetManagerRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "total_fleet_manager_requests",
		Help: "The total number of requests send to fleet-manager",
	})

	fleetManagerRequestErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "total_fleet_manager_request_errors",
		Help: "The total number of request errors for requests send to fleet-manager",
	})
}
