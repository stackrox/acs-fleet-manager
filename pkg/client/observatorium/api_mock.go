package observatorium

import (
	"context"
	"fmt"
	"strings"
	"time"

	pV1 "github.com/prometheus/client_golang/api/prometheus/v1"
	pModel "github.com/prometheus/common/model"
)

// MockAPI ...
func (c *Client) MockAPI() pV1.API {
	return &httpAPIMock{}
}

type httpAPIMock struct{}

// Query performs a query for the dinosaur metrics.
func (t *httpAPIMock) Query(ctx context.Context, query string, ts time.Time, opts ...pV1.Option) (pModel.Value, pV1.Warnings, error) {
	values := getMockQueryData(query)
	return values, []string{}, nil
}

// QueryRange(ctx context.Context, query string, r pV1.Range) (pModel.Value, pV1.Warnings, error) Performs a query range for the dinosaur metrics
func (*httpAPIMock) QueryRange(ctx context.Context, query string, r pV1.Range, opts ...pV1.Option) (pModel.Value, pV1.Warnings, error) {
	values := getMockQueryRangeData(query)
	return values, []string{}, nil
}

// Alerts Not implemented
func (*httpAPIMock) Alerts(ctx context.Context) (pV1.AlertsResult, error) {
	return pV1.AlertsResult{}, fmt.Errorf("not implemented")
}

// AlertManagers ...
func (*httpAPIMock) AlertManagers(ctx context.Context) (pV1.AlertManagersResult, error) {
	return pV1.AlertManagersResult{}, fmt.Errorf("not implemented")
}

// CleanTombstones ...
func (*httpAPIMock) CleanTombstones(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}

// Config ...
func (*httpAPIMock) Config(ctx context.Context) (pV1.ConfigResult, error) {
	return pV1.ConfigResult{}, fmt.Errorf("not implemented")
}

// DeleteSeries ...
func (*httpAPIMock) DeleteSeries(ctx context.Context, matches []string, startTime time.Time, endTime time.Time) error {
	return fmt.Errorf("not implemented")
}

// Flags ...
func (*httpAPIMock) Flags(ctx context.Context) (pV1.FlagsResult, error) {
	return pV1.FlagsResult{}, fmt.Errorf("not implemented")
}

// LabelNames ...
func (*httpAPIMock) LabelNames(ctx context.Context, matches []string, startTime time.Time, endTime time.Time) ([]string, pV1.Warnings, error) {
	return []string{}, pV1.Warnings{}, fmt.Errorf("not implemented")
}

// LabelValues ...
func (*httpAPIMock) LabelValues(ctx context.Context, label string, matches []string, startTime time.Time, endTime time.Time) (pModel.LabelValues, pV1.Warnings, error) {
	return pModel.LabelValues{}, pV1.Warnings{}, fmt.Errorf("not implemented")
}

// Series ...
func (*httpAPIMock) Series(ctx context.Context, matches []string, startTime time.Time, endTime time.Time) ([]pModel.LabelSet, pV1.Warnings, error) {
	return []pModel.LabelSet{}, pV1.Warnings{}, fmt.Errorf("not implemented")
}

// Snapshot ...
func (*httpAPIMock) Snapshot(ctx context.Context, skipHead bool) (pV1.SnapshotResult, error) {
	return pV1.SnapshotResult{}, fmt.Errorf("not implemented")
}

// Rules ...
func (*httpAPIMock) Rules(ctx context.Context) (pV1.RulesResult, error) {
	return pV1.RulesResult{}, fmt.Errorf("not implemented")
}

// Targets ...
func (*httpAPIMock) Targets(ctx context.Context) (pV1.TargetsResult, error) {
	return pV1.TargetsResult{}, fmt.Errorf("not implemented")
}

// TargetsMetadata ...
func (*httpAPIMock) TargetsMetadata(ctx context.Context, matchTarget string, metric string, limit string) ([]pV1.MetricMetadata, error) {
	return []pV1.MetricMetadata{}, fmt.Errorf("not implemented")
}

// Metadata ...
func (*httpAPIMock) Metadata(ctx context.Context, metric string, limit string) (map[string][]pV1.Metadata, error) {
	return nil, fmt.Errorf("not implemented")
}

// TSDB ...
func (*httpAPIMock) TSDB(ctx context.Context) (pV1.TSDBResult, error) {
	return pV1.TSDBResult{}, fmt.Errorf("not implemented")
}

// Runtimeinfo ...
func (*httpAPIMock) Runtimeinfo(ctx context.Context) (pV1.RuntimeinfoResult, error) {
	return pV1.RuntimeinfoResult{}, fmt.Errorf("not implemented")
}

// Buildinfo ...
func (*httpAPIMock) Buildinfo(ctx context.Context) (pV1.BuildinfoResult, error) {
	return pV1.BuildinfoResult{}, fmt.Errorf("not implemented")
}

// QueryExemplars ...
func (*httpAPIMock) QueryExemplars(ctx context.Context, query string, startTime time.Time, endTime time.Time) ([]pV1.ExemplarQueryResult, error) {
	return []pV1.ExemplarQueryResult{}, fmt.Errorf("not implemented")
}

// WalReplay ...
func (*httpAPIMock) WalReplay(ctx context.Context) (pV1.WalReplayStatus, error) {
	return pV1.WalReplayStatus{}, fmt.Errorf("not implemented")
}

// getMockQueryData
func getMockQueryData(query string) pModel.Vector {
	for key, values := range queryData {
		if strings.Contains(query, key) {
			return values
		}
	}
	return pModel.Vector{}
}

// getMockQueryRangeData
func getMockQueryRangeData(query string) pModel.Matrix {
	for key, values := range rangeQuerydata {
		if strings.Contains(query, key) {
			return values
		}
	}
	return pModel.Matrix{}
}

var rangeQuerydata = map[string]pModel.Matrix{
	"kubelet_volume_stats_available_bytes": {
		fakeMetricData("kubelet_volume_stats_available_bytes", 220792516608),
	},
	"dinosaur_server_brokertopicmetrics_messages_in_total": {
		fakeMetricData("dinosaur_server_brokertopicmetrics_messages_in_total", 3040),
	},
	"dinosaur_server_brokertopicmetrics_bytes_in_total": {
		fakeMetricData("dinosaur_server_brokertopicmetrics_bytes_in_total", 293617),
	},
	"dinosaur_server_brokertopicmetrics_bytes_out_total": {
		fakeMetricData("dinosaur_server_brokertopicmetrics_bytes_out_total", 152751),
	},
	"dinosaur_controller_dinosaurcontroller_offline_partitions_count": {
		fakeMetricData("dinosaur_controller_dinosaurcontroller_offline_partitions_count", 0),
	},
	"dinosaur_controller_dinosaurcontroller_global_partition_count": {
		fakeMetricData("dinosaur_controller_dinosaurcontroller_global_partition_count", 0),
	},
	"dinosaur_broker_quota_softlimitbytes": {
		fakeMetricData("dinosaur_broker_quota_softlimitbytes", 10000),
	},
	"dinosaur_broker_quota_totalstorageusedbytes": {
		fakeMetricData("dinosaur_broker_quota_totalstorageusedbytes", 1237582),
	},
	"dinosaur_topic:dinosaur_log_log_size:sum": {
		fakeMetricData("dinosaur_topic:dinosaur_log_log_size:sum", 220),
	},
	"dinosaur_namespace:dinosaur_server_socket_server_metrics_connection_creation_rate:sum": {
		fakeMetricData("dinosaur_namespace:dinosaur_server_socket_server_metrics_connection_creation_rate:sum", 20),
	},
	"dinosaur_topic:dinosaur_topic_partitions:sum": {
		fakeMetricData("dinosaur_topic:dinosaur_topic_partitions:sum", 20),
	},
	"dinosaur_topic:dinosaur_topic_partitions:count": {
		fakeMetricData("dinosaur_topic:dinosaur_topic_partitions:count", 20),
	},
	"consumergroup:dinosaur_consumergroup_members:count": {
		fakeMetricData("consumergroup:dinosaur_consumergroup_members:count", 20),
	},
	"dinosaur_namespace:dinosaur_server_socket_server_metrics_connection_count:sum": {
		fakeMetricData("dinosaur_namespace:dinosaur_server_socket_server_metrics_connection_count:sum", 20),
	},
}

func fakeMetricData(name string, value int) *pModel.SampleStream {
	return &pModel.SampleStream{
		Metric: pModel.Metric{
			"__name__":                     pModel.LabelValue(name),
			"pod":                          "whatever",
			"dinosaur_operator_io_cluster": "whatever",
		},
		Values: []pModel.SamplePair{
			{Timestamp: 0, Value: pModel.SampleValue(value)},
			{Timestamp: 0, Value: pModel.SampleValue(value)},
		},
	}
}

var queryData = map[string]pModel.Vector{
	"dinosaur_operator_resource_state": {
		&pModel.Sample{
			Metric: pModel.Metric{
				"dinosaur_operator_io_kind": "Dinosaur",
				"dinosaur_operator_io_name": "test-dinosaur",
				"namespace":                 "my-dinosaur-namespace",
			},
			Timestamp: pModel.Time(1607506882175),
			Value:     1,
		},
	},

	"dinosaur_server_brokertopicmetrics_bytes_in_total": {
		&pModel.Sample{
			Metric: pModel.Metric{
				"__name__":                     "dinosaur_server_brokertopicmetrics_bytes_in_total",
				"pod":                          "whatever",
				"dinosaur_operator_io_cluster": "whatever",
				"topic":                        "whatever",
			},
			Timestamp: pModel.Time(1607506882175),
			Value:     293617,
		},
	},
	"dinosaur_server_brokertopicmetrics_messages_in_total": {
		&pModel.Sample{
			Metric: pModel.Metric{
				"__name__":                     "dinosaur_server_brokertopicmetrics_messages_in_total",
				"pod":                          "whatever",
				"dinosaur_operator_io_cluster": "whatever",
				"topic":                        "whatever",
			},
			Timestamp: pModel.Time(1607506882175),
			Value:     1016,
		},
	},
	"dinosaur_broker_quota_softlimitbytes": {
		&pModel.Sample{
			Metric: pModel.Metric{
				"__name__":                     "dinosaur_broker_quota_softlimitbytes",
				"pod":                          "whatever",
				"dinosaur_operator_io_cluster": "whatever",
				"topic":                        "whatever",
			},
			Timestamp: pModel.Time(1607506882175),
			Value:     30000,
		},
	},
	"dinosaur_broker_quota_totalstorageusedbytes": {
		&pModel.Sample{
			Metric: pModel.Metric{
				"__name__":                     "dinosaur_broker_quota_totalstorageusedbytes",
				"pod":                          "whatever",
				"dinosaur_operator_io_cluster": "whatever",
				"topic":                        "whatever",
			},
			Timestamp: pModel.Time(1607506882175),
			Value:     2207924332,
		},
	},
	"kubelet_volume_stats_available_bytes": {
		&pModel.Sample{
			Metric: pModel.Metric{
				"__name__":              "kubelet_volume_stats_available_bytes",
				"persistentvolumeclaim": "whatever",
			},
			Timestamp: pModel.Time(1607506882175),
			Value:     220792492032,
		},
	},
}
