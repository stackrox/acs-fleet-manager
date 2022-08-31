package fleetshardmetrics

import "github.com/prometheus/client_golang/prometheus"

const metricsPrefix = "acs_fleetshard_"

var (
	k8sRequests               prometheus.Counter
	k8sRequestErrors          prometheus.Counter
	fleetManagerRequests      prometheus.Counter
	fleetManagerRequestErrors prometheus.Counter
)

func init() {
	initMetrics()
}

func initMetrics() {
	k8sRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Name: metricsPrefix + "total_k8s_requests",
		Help: "The total number of requests send to the target kubernetes cluster",
	})

	k8sRequestErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: metricsPrefix + "total_k8s_request_errors",
		Help: "The total number of unexpected errors for requests send to the target kubernetes cluster",
	})

	fleetManagerRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Name: metricsPrefix + "total_fleet_manager_requests",
		Help: "The total number of requests send to fleet-manager",
	})

	fleetManagerRequestErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: metricsPrefix + "total_fleet_manager_request_errors",
		Help: "The total number of request errors for requests send to fleet-manager",
	})
}

func registerCustomMetrics(r prometheus.Registerer) {
	r.MustRegister(k8sRequestErrors)
	r.MustRegister(k8sRequests)
	r.MustRegister(fleetManagerRequestErrors)
	r.MustRegister(fleetManagerRequests)
}

// IncrementK8sRequests increments the metric counter for k8s requests
func IncrementK8sRequests() {
	k8sRequests.Inc()
}

// IncrementK8sRequestErrors increments the metric counter for k8s request errors
func IncrementK8sRequestErrors() {
	k8sRequestErrors.Inc()
}

// IncrementFleetManagerRequests increments the metric counter for fleet-manager requests
func IncrementFleetManagerRequests() {
	fleetManagerRequests.Inc()
}

// IncrementFleetManagerRequestErrors increments the metric counter for fleet-manager request errors
func IncrementFleetManagerRequestErrors() {
	fleetManagerRequestErrors.Inc()
}
