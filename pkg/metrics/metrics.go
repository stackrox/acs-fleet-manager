package metrics

import (
	"strconv"
	"time"

	constants2 "github.com/stackrox/acs-fleet-manager/internal/central/constants"

	"github.com/stackrox/acs-fleet-manager/pkg/api"

	"github.com/prometheus/client_golang/prometheus"
)

// FleetManager ...
const (
	// FleetManager - metrics prefix
	FleetManager = "acs_fleet_manager"

	// ClusterCreateRequestDuration - name of cluster creation duration metric
	ClusterCreateRequestDuration = "worker_cluster_duration"
	// CentralCreateRequestDuration - name of central creation duration metric
	CentralCreateRequestDuration = "worker_central_duration"

	labelJobType           = "jobType"
	LabelID                = "id"
	LabelStatus            = "status"
	LabelClusterID         = "cluster_id"
	LabelClusterExternalID = "external_id"

	// CentralOperationsSuccessCount - name of the metric for Central-related successful operations
	CentralOperationsSuccessCount = "central_operations_success_count"
	// CentralOperationsTotalCount - name of the metric for all Central-related operations
	CentralOperationsTotalCount = "central_operations_total_count"

	// CentralRequestsStatus - central requests status metric
	CentralRequestsStatusSinceCreated = "central_requests_status_since_created_in_seconds"
	CentralRequestsStatusCount        = "central_requests_status_count"

	// ClusterOperationsSuccessCount - name of the metric for cluster-related successful operations
	ClusterOperationsSuccessCount = "cluster_operations_success_count"
	// ClusterOperationsTotalCount - name of the metric for all cluster-related operations
	ClusterOperationsTotalCount = "cluster_operations_total_count"
	labelOperation              = "operation"

	ReconcilerDuration     = "reconciler_duration_in_seconds"
	ReconcilerSuccessCount = "reconciler_success_count"
	ReconcilerFailureCount = "reconciler_failure_count"
	ReconcilerErrorsCount  = "reconciler_errors_count"
	labelWorkerType        = "worker_type"

	ClusterStatusSinceCreated = "cluster_status_since_created_in_seconds"
	ClusterStatusCount        = "cluster_status_count"

	CentralPerClusterCount = "central_per_cluster_count"

	LeaderWorker = "leader_worker"

	// ObservatoriumRequestCount - metric name for the number of observatorium requests sent
	ObservatoriumRequestCount = "observatorium_request_count"
	// ObservatoriumRequestDuration - metric name for observatorium request duration in seconds
	ObservatoriumRequestDuration = "observatorium_request_duration"

	// DatabaseQueryCount - metric name for the number of database query sent
	DatabaseQueryCount = "database_query_count"
	// DatabaseQueryDuration - metric name for database query duration in milliseconds
	DatabaseQueryDuration = "database_query_duration"

	// ClusterStatusMaxCapacity - metric name for the maximum central instance capacity
	ClusterStatusCapacityMax = "cluster_status_capacity_max"

	// ClusterStatusCapacityUsed - metric name for the current number of instances
	ClusterStatusCapacityUsed = "cluster_status_capacity_used"

	LabelStatusCode = "code"
	LabelMethod     = "method"
	LabelPath       = "path"

	LabelDatabaseQueryStatus = "status"
	LabelDatabaseQueryType   = "query"
	LabelRegion              = "region"
	LabelInstanceType        = "instance_type"
)

// JobType metric to capture
type JobType string

// JobTypeClusterCreate ...
var (
	// JobTypeClusterCreate - cluster_create job type
	JobTypeClusterCreate JobType = "cluster_create"
	// JobTypeCentralCreate - central_create job type
	JobTypeCentralCreate JobType = "central_create"
)

// JobsMetricsLabels is the slice of labels to add to job metrics
var JobsMetricsLabels = []string{
	labelJobType,
}

// centralStatusSinceCreatedMetricLabels  is the slice of labels to add to
var centralStatusSinceCreatedMetricLabels = []string{
	LabelStatus,
	LabelID,
	LabelClusterID,
}

// centralStatusCountMetricLabels  is the slice of labels to add to
var centralStatusCountMetricLabels = []string{
	LabelStatus,
}

// CentralOperationsCountMetricsLabels - is the slice of labels to add to Central operations count metrics
var CentralOperationsCountMetricsLabels = []string{
	labelOperation,
}

// CentralPerClusterCountMetricsLabels ...
var CentralPerClusterCountMetricsLabels = []string{
	LabelClusterID,
	LabelClusterExternalID,
}

// ClusterOperationsCountMetricsLabels - is the slice of labels to add to Central operations count metrics
var ClusterOperationsCountMetricsLabels = []string{
	labelOperation,
}

// ClusterStatusSinceCreatedMetricsLabels ...
var ClusterStatusSinceCreatedMetricsLabels = []string{
	LabelID,
	LabelClusterID,
	LabelStatus,
}

// ClusterStatusCountMetricsLabels ...
var ClusterStatusCountMetricsLabels = []string{
	LabelStatus,
}

// ReconcilerMetricsLabels ...
var ReconcilerMetricsLabels = []string{
	labelWorkerType,
}

var observatoriumRequestMetricsLabels = []string{
	LabelStatusCode,
	LabelMethod,
	LabelPath,
}

// DatabaseMetricsLabels ...
var DatabaseMetricsLabels = []string{
	LabelDatabaseQueryStatus,
	LabelDatabaseQueryType,
}

var clusterStatusCapacityLabels = []string{
	LabelRegion,
	LabelInstanceType,
	LabelClusterID,
}

// #### Metrics for Dataplane clusters - Start ####
// create a new histogramVec for cluster creation duration
var requestClusterCreationDurationMetric = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Subsystem: FleetManager,
		Name:      ClusterCreateRequestDuration,
		Help:      "Cluster creation duration in seconds.",
		Buckets: []float64{
			1800.0,
			2400.0,
			3600.0,
			4800.0,
			7200.0,
			10800.0,
		},
	},
	JobsMetricsLabels,
)

// UpdateClusterCreationDurationMetric records the duration of a job type
func UpdateClusterCreationDurationMetric(jobType JobType, elapsed time.Duration) {
	labels := prometheus.Labels{
		labelJobType: string(jobType),
	}
	requestClusterCreationDurationMetric.With(labels).Observe(elapsed.Seconds())
}

// create a new counterVec for successful cluster operation counts
var clusterOperationsSuccessCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: FleetManager,
		Name:      ClusterOperationsSuccessCount,
		Help:      "number of successful cluster operations",
	},
	ClusterOperationsCountMetricsLabels,
)

// IncreaseClusterSuccessOperationsCountMetric - increase counter for clusterOperationsSuccessCountMetric
func IncreaseClusterSuccessOperationsCountMetric(operation constants2.ClusterOperation) {
	labels := prometheus.Labels{
		labelOperation: operation.String(),
	}
	clusterOperationsSuccessCountMetric.With(labels).Inc()
}

// UpdateClusterStatusCapacityMaxCount - sets maximum capacity per region and instance type
func UpdateClusterStatusCapacityMaxCount(region, instanceType, clusterID string, count float64) {
	labels := prometheus.Labels{
		LabelRegion:       region,
		LabelInstanceType: instanceType,
		LabelClusterID:    clusterID,
	}
	clusterStatusCapacityMaxMetric.With(labels).Set(count)
}

// UpdateClusterStatusCapacityUsedCount - sets used capacity per region and instance type
func UpdateClusterStatusCapacityUsedCount(region, instanceType, clusterID string, count float64) {
	labels := prometheus.Labels{
		LabelRegion:       region,
		LabelInstanceType: instanceType,
		LabelClusterID:    clusterID,
	}
	clusterStatusCapacityUsedMetric.With(labels).Set(count)
}

// create a new counterVec for total cluster operation counts
var clusterOperationsTotalCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: FleetManager,
		Name:      ClusterOperationsTotalCount,
		Help:      "number of total cluster operations",
	},
	ClusterOperationsCountMetricsLabels,
)

// create a new gaugeVec for the maximum central instance capacity per region
var clusterStatusCapacityMaxMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: FleetManager,
		Name:      ClusterStatusCapacityMax,
		Help:      "number of allowed instances per region and instance type",
	},
	clusterStatusCapacityLabels,
)

// create a new gauge vec fot the number of central instances grouped by region and instance type
var clusterStatusCapacityUsedMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: FleetManager,
		Name:      ClusterStatusCapacityUsed,
		Help:      "number of existing instances per region and instance type",
	},
	clusterStatusCapacityLabels,
)

// IncreaseClusterTotalOperationsCountMetric - increase counter for clusterOperationsTotalCountMetric
func IncreaseClusterTotalOperationsCountMetric(operation constants2.ClusterOperation) {
	labels := prometheus.Labels{
		labelOperation: operation.String(),
	}
	clusterOperationsTotalCountMetric.With(labels).Inc()
}

// create a new GaugeVec for cluster status since created
var clusterStatusSinceCreatedMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: FleetManager,
		Name:      ClusterStatusSinceCreated,
		Help:      "metrics to track the status of a dataplane cluster and how long since it's been created",
	},
	ClusterStatusSinceCreatedMetricsLabels,
)

// UpdateClusterStatusSinceCreatedMetric ...
func UpdateClusterStatusSinceCreatedMetric(cluster api.Cluster, status api.ClusterStatus) {
	labels := prometheus.Labels{
		LabelStatus:    string(status),
		LabelID:        cluster.ID,
		LabelClusterID: cluster.ClusterID,
	}
	clusterStatusSinceCreatedMetric.With(labels).Set(time.Since(cluster.CreatedAt).Seconds())
}

var clusterStatusCountMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: FleetManager,
		Name:      ClusterStatusCount,
		Help:      "metrics to record the number of Dataplane clusters in each status",
	},
	ClusterStatusCountMetricsLabels,
)

// UpdateClusterStatusCountMetric ...
func UpdateClusterStatusCountMetric(status api.ClusterStatus, count int) {
	labels := prometheus.Labels{
		LabelStatus: string(status),
	}
	clusterStatusCountMetric.With(labels).Set(float64(count))
}

var centralPerClusterCountMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: FleetManager,
		Name:      CentralPerClusterCount,
		Help:      "the number of Central instances per data plane cluster",
	},
	CentralPerClusterCountMetricsLabels)

// UpdateCentralPerClusterCountMetric ...
func UpdateCentralPerClusterCountMetric(clusterID string, clusterExternalID string, count int) {
	labels := prometheus.Labels{
		LabelClusterID:         clusterID,
		LabelClusterExternalID: clusterExternalID,
	}
	centralPerClusterCountMetric.With(labels).Set(float64(count))
}

// #### Metrics for Dataplane clusters - End ####

// #### Metrics for Centrals - Start ####
// create a new histogramVec for central creation duration
var requestCentralCreationDurationMetric = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Subsystem: FleetManager,
		Name:      CentralCreateRequestDuration,
		Help:      "Central creation duration in seconds.",
		Buckets: []float64{
			1.0,
			30.0,
			60.0,
			120.0,
			150.0,
			180.0,
			210.0,
			240.0,
			270.0,
			300.0,
			330.0,
			360.0,
			390.0,
			420.0,
			450.0,
			480.0,
			510.0,
			540.0,
			570.0,
			600.0,
			900.0,
			1200.0,
			1800.0,
			2400.0,
			3600.0,
			4800.0,
			7200.0,
		},
	},
	JobsMetricsLabels,
)

// UpdateCentralCreationDurationMetric records the duration of a job type
func UpdateCentralCreationDurationMetric(jobType JobType, elapsed time.Duration) {
	labels := prometheus.Labels{
		labelJobType: string(jobType),
	}
	requestCentralCreationDurationMetric.With(labels).Observe(elapsed.Seconds())
}

// create a new counterVec for Central operations counts
var centralOperationsSuccessCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: FleetManager,
		Name:      CentralOperationsSuccessCount,
		Help:      "number of successful central operations",
	},
	CentralOperationsCountMetricsLabels,
)

// UpdateCentralRequestsStatusSinceCreatedMetric ...
func UpdateCentralRequestsStatusSinceCreatedMetric(status constants2.CentralStatus, centralID string, clusterID string, elapsed time.Duration) {
	labels := prometheus.Labels{
		LabelStatus:    string(status),
		LabelID:        centralID,
		LabelClusterID: clusterID,
	}
	centralStatusSinceCreatedMetric.With(labels).Set(elapsed.Seconds())
}

// UpdateCentralRequestsStatusCountMetric ...
func UpdateCentralRequestsStatusCountMetric(status constants2.CentralStatus, count int) {
	labels := prometheus.Labels{
		LabelStatus: string(status),
	}
	CentralStatusCountMetric.With(labels).Set(float64(count))
}

// CentralStatusCountMetric create a new GaugeVec for status counts
var CentralStatusCountMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: FleetManager,
		Name:      CentralRequestsStatusCount,
		Help:      "number of total Central instances in each status",
	},
	centralStatusCountMetricLabels,
)

// create a new GaugeVec for centrals status duration
var centralStatusSinceCreatedMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: FleetManager,
		Name:      CentralRequestsStatusSinceCreated,
		Help:      "metrics to track the status of a Central instance and how long since it's been created",
	},
	centralStatusSinceCreatedMetricLabels,
)

// IncreaseCentralSuccessOperationsCountMetric - increase counter for the centralOperationsSuccessCountMetric
func IncreaseCentralSuccessOperationsCountMetric(operation constants2.CentralOperation) {
	labels := prometheus.Labels{
		labelOperation: operation.String(),
	}
	centralOperationsSuccessCountMetric.With(labels).Inc()
}

// create a new counterVec for total Central operations counts
var centralOperationsTotalCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: FleetManager,
		Name:      CentralOperationsTotalCount,
		Help:      "number of total central operations",
	},
	CentralOperationsCountMetricsLabels,
)

// IncreaseCentralTotalOperationsCountMetric - increase counter for the centralOperationsTotalCountMetric
func IncreaseCentralTotalOperationsCountMetric(operation constants2.CentralOperation) {
	labels := prometheus.Labels{
		labelOperation: operation.String(),
	}
	centralOperationsTotalCountMetric.With(labels).Inc()
}

// #### Metrics for Centrals - End ####

// #### Metrics for Reconcilers - Start ####
// create a new gaugeVec for reconciler duration
var reconcilerDurationMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: FleetManager,
		Name:      ReconcilerDuration,
		Help:      "Duration of each background reconcile in seconds.",
	},
	ReconcilerMetricsLabels,
)

// UpdateReconcilerDurationMetric ...
func UpdateReconcilerDurationMetric(reconcilerType string, elapsed time.Duration) {
	labels := prometheus.Labels{
		labelWorkerType: reconcilerType,
	}
	reconcilerDurationMetric.With(labels).Set(float64(elapsed.Seconds()))
}

var reconcilerSuccessCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: FleetManager,
		Name:      ReconcilerSuccessCount,
		Help:      "count of success operations of the backgroup reconcilers",
	}, ReconcilerMetricsLabels)

// IncreaseReconcilerSuccessCount ...
func IncreaseReconcilerSuccessCount(reconcilerType string) {
	labels := prometheus.Labels{
		labelWorkerType: reconcilerType,
	}
	reconcilerSuccessCountMetric.With(labels).Inc()
}

var reconcilerFailureCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: FleetManager,
		Name:      ReconcilerFailureCount,
		Help:      "count of failed operations of the backgroup reconcilers",
	}, ReconcilerMetricsLabels)

// IncreaseReconcilerFailureCount ...
func IncreaseReconcilerFailureCount(reconcilerType string) {
	labels := prometheus.Labels{
		labelWorkerType: reconcilerType,
	}
	reconcilerFailureCountMetric.With(labels).Inc()
}

var reconcilerErrorsCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: FleetManager,
		Name:      ReconcilerErrorsCount,
		Help:      "count of errors occured during backgroup reconcilers runs",
	}, ReconcilerMetricsLabels)

// IncreaseReconcilerErrorsCount ...
func IncreaseReconcilerErrorsCount(reconcilerType string, numOfErr int) {
	labels := prometheus.Labels{
		labelWorkerType: reconcilerType,
	}
	reconcilerErrorsCountMetric.With(labels).Add(float64(numOfErr))
}

var leaderWorkerMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: FleetManager,
		Name:      LeaderWorker,
		Help:      "metrics to indicate if the current process is the leader among the workers",
	}, ReconcilerMetricsLabels)

// SetLeaderWorkerMetric will set the metric value to 1 if the worker is the leader, and 0 if the worker is not the leader.
// Then when the metrics is scraped, Prometheus will add additional information like pod name, which then can be used to display which pod is the leader.
func SetLeaderWorkerMetric(workerType string, leader bool) {
	labels := prometheus.Labels{
		labelWorkerType: workerType,
	}
	val := 0
	if leader {
		val = 1
	}
	leaderWorkerMetric.With(labels).Set(float64(val))
}

// #### Metrics for Reconcilers - End ####

// #### Metrics for Observatorium ####

// register observatorium request count metric
//
//	observatorium_request_count - Number of Observatorium requests sent partitioned by http status code, method and url path
var observatoriumRequestCountMetric = prometheus.NewCounterVec(prometheus.CounterOpts{
	Subsystem: FleetManager,
	Name:      ObservatoriumRequestCount,
	Help:      "number of requests sent to Observatorium. If no response was received, the value of code should be '0' (this can happen on request timeout or failure to connect to Observatorium).",
}, observatoriumRequestMetricsLabels)

// IncreaseObservatoriumRequestCount Increase the observatorium request count metric with the following labels:
//   - code: HTTP Status code (i.e. 200 or 500)
//   - path: Request URL path (i.e. /api/v1/query)
//   - method: HTTP Method (i.e. GET or POST)
func IncreaseObservatoriumRequestCount(code int, path, method string) {
	labels := prometheus.Labels{
		LabelStatusCode: strconv.Itoa(code),
		LabelPath:       path,
		LabelMethod:     method,
	}
	observatoriumRequestCountMetric.With(labels).Inc()
}

// register observatorium request duration metric. Each metric is partitioned by http status code, method and url path
//
//	observatorium_request_duration_sum - Total time to send requests to Observatorium in seconds.
//	observatorium_request_duration_count - Total number of Observatorium requests measured.
//	observatorium_request_duration_bucket - Number of Observatorium requests organized in buckets.
var observatoriumRequestDurationMetric = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Subsystem: FleetManager,
		Name:      ObservatoriumRequestDuration,
		Help:      `Observatorium request duration in seconds. If no response was received, the value of code should be '0' (this can happen on request timeout or failure to connect to Observatorium).`,
		Buckets: []float64{
			0.1,
			1.0,
			10.0,
			30.0,
		},
	},
	observatoriumRequestMetricsLabels,
)

// UpdateObservatoriumRequestDurationMetric Update the observatorium request duration metric with the following labels:
//   - code: HTTP Status code (i.e. 200 or 500)
//   - path: Request url path (i.e. /api/v1/query)
//   - method: HTTP Method (i.e. GET or POST)
func UpdateObservatoriumRequestDurationMetric(code int, path, method string, elapsed time.Duration) {
	labels := prometheus.Labels{
		LabelStatusCode: strconv.Itoa(code),
		LabelPath:       path,
		LabelMethod:     method,
	}
	observatoriumRequestDurationMetric.With(labels).Observe(elapsed.Seconds())
}

// #### Metrics for Observatorium - End ####

// #### Metrics for Database ####

// register database query count metric
//
//	database_query_count - Number of Database query sent partitioned by status, and sql query type
var databaseRequestCountMetric = prometheus.NewCounterVec(prometheus.CounterOpts{
	Subsystem: FleetManager,
	Name:      DatabaseQueryCount,
	Help:      "number of query sent to Dataase.",
}, DatabaseMetricsLabels)

// IncreaseDatabaseQueryCount Increase the database query count metric with the following labels:
//   - status: (i.e. "success" or "failure")
//   - queryType: (i.e. "SELECT", "UPDATE", "INSERT", "DELETE")
func IncreaseDatabaseQueryCount(status string, queryType string) {
	labels := prometheus.Labels{
		LabelDatabaseQueryStatus: status,
		LabelDatabaseQueryType:   queryType,
	}
	databaseRequestCountMetric.With(labels).Inc()
}

// register database query duration metric. Each metric is partitioned by status, query type
//
//	database_query_duration_sum - Total time to send requests to Database in milliseconds.
//	database_query_duration_count - Total number of database query measured.
//	database_query_duration_bucket - Number of Database queries organized in buckets.
var databaseQueryDurationMetric = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Subsystem: FleetManager,
		Name:      DatabaseQueryDuration,
		Help:      `Database query duration in milliseconds.`,
		Buckets: []float64{
			1.0,
			30.0,
			60.0,
			120.0,
			150.0,
			180.0,
			210.0,
			240.0,
			270.0,
			300.0,
			330.0,
			360.0,
			390.0,
			420.0,
			450.0,
			480.0,
			510.0,
			540.0,
			570.0,
			600.0,
			900.0,
			1200.0,
			1800.0,
			2400.0,
			3600.0,
			4800.0,
			7200.0,
		},
	},
	DatabaseMetricsLabels,
)

// UpdateDatabaseQueryDurationMetric Update the observatorium request duration metric with the following labels:
//   - status: (i.e. "success" or "failure")
//   - queryType: (i.e. "SELECT", "UPDATE", "INSERT", "DELETE")
func UpdateDatabaseQueryDurationMetric(status string, queryType string, elapsed time.Duration) {
	labels := prometheus.Labels{
		LabelDatabaseQueryStatus: status,
		LabelDatabaseQueryType:   queryType,
	}
	databaseQueryDurationMetric.With(labels).Observe(float64(elapsed.Milliseconds()))
}

// #### Metrics for Database - End ####

// register the metric(s)
func init() {
	// metrics for data plane clusters
	prometheus.MustRegister(requestClusterCreationDurationMetric)
	prometheus.MustRegister(clusterOperationsSuccessCountMetric)
	prometheus.MustRegister(clusterOperationsTotalCountMetric)
	prometheus.MustRegister(clusterStatusSinceCreatedMetric)
	prometheus.MustRegister(clusterStatusCountMetric)
	prometheus.MustRegister(centralPerClusterCountMetric)
	prometheus.MustRegister(clusterStatusCapacityMaxMetric)
	prometheus.MustRegister(clusterStatusCapacityUsedMetric)

	// metrics for Centrals
	prometheus.MustRegister(requestCentralCreationDurationMetric)
	prometheus.MustRegister(centralOperationsSuccessCountMetric)
	prometheus.MustRegister(centralOperationsTotalCountMetric)
	prometheus.MustRegister(centralStatusSinceCreatedMetric)
	prometheus.MustRegister(CentralStatusCountMetric)

	// metrics for reconcilers
	prometheus.MustRegister(reconcilerDurationMetric)
	prometheus.MustRegister(reconcilerSuccessCountMetric)
	prometheus.MustRegister(reconcilerFailureCountMetric)
	prometheus.MustRegister(reconcilerErrorsCountMetric)
	prometheus.MustRegister(leaderWorkerMetric)

	// metrics for observatorium
	prometheus.MustRegister(observatoriumRequestCountMetric)
	prometheus.MustRegister(observatoriumRequestDurationMetric)

	// metrics for database
	prometheus.MustRegister(databaseRequestCountMetric)
	prometheus.MustRegister(databaseQueryDurationMetric)
}

// ResetMetricsForCentralManagers will reset the metrics for the CentralManager background reconciler
// This is needed because if current process is not the leader anymore, the metrics need to be reset otherwise staled data will be scraped
func ResetMetricsForCentralManagers() {
	centralStatusSinceCreatedMetric.Reset()
	CentralStatusCountMetric.Reset()
}

// ResetMetricsForClusterManagers will reset the metrics for the ClusterManager background reconciler
// This is needed because if current process is not the leader anymore, the metrics need to be reset otherwise staled data will be scraped
func ResetMetricsForClusterManagers() {
	clusterStatusSinceCreatedMetric.Reset()
	clusterStatusCountMetric.Reset()
	centralPerClusterCountMetric.Reset()
	clusterStatusCapacityMaxMetric.Reset()
	clusterStatusCapacityUsedMetric.Reset()
}

// ResetMetricsForReconcilers will reset the metrics related to the reconcilers
// This is needed because if current process is not the leader anymore, the metrics need to be reset otherwise staled data will be scraped
func ResetMetricsForReconcilers() {
	reconcilerDurationMetric.Reset()
	reconcilerSuccessCountMetric.Reset()
	reconcilerFailureCountMetric.Reset()
	reconcilerErrorsCountMetric.Reset()
}

// ResetMetricsForObservatorium will reset the metrics related to Observatorium requests
// This is needed because if current process is not the leader anymore, the metrics need to be reset otherwise staled data will be scraped
func ResetMetricsForObservatorium() {
	observatoriumRequestCountMetric.Reset()
	observatoriumRequestDurationMetric.Reset()
}

// Reset the metrics we have defined. It is mainly used for testing.
func Reset() {
	requestClusterCreationDurationMetric.Reset()
	clusterOperationsSuccessCountMetric.Reset()
	clusterOperationsTotalCountMetric.Reset()
	clusterStatusSinceCreatedMetric.Reset()
	clusterStatusCountMetric.Reset()
	centralPerClusterCountMetric.Reset()
	clusterStatusCapacityMaxMetric.Reset()
	clusterStatusCapacityUsedMetric.Reset()

	requestCentralCreationDurationMetric.Reset()
	centralOperationsSuccessCountMetric.Reset()
	centralOperationsTotalCountMetric.Reset()
	centralStatusSinceCreatedMetric.Reset()
	CentralStatusCountMetric.Reset()

	reconcilerDurationMetric.Reset()
	reconcilerSuccessCountMetric.Reset()
	reconcilerFailureCountMetric.Reset()
	reconcilerErrorsCountMetric.Reset()
	leaderWorkerMetric.Reset()

	ResetMetricsForObservatorium()

	databaseRequestCountMetric.Reset()
	databaseQueryDurationMetric.Reset()
}
