package observatorium

import (
	"context"
	"strings"
	"time"

	pV1 "github.com/prometheus/client_golang/api/prometheus/v1"
	pModel "github.com/prometheus/common/model"
)

// API an alias for pV1.API
//go:generate moq -out api_moq.go . API
type API = pV1.API

// MockAPI returns a mocked instance of pV1.API
func (c *Client) MockAPI() pV1.API {
	return &APIMock{
		QueryFunc: func(ctx context.Context, query string, ts time.Time, opts ...pV1.Option) (pModel.Value, pV1.Warnings, error) {
			values := getMockQueryData(query)
			return values, []string{}, nil
		},
		QueryRangeFunc: func(ctx context.Context, query string, r pV1.Range, opts ...pV1.Option) (pModel.Value, pV1.Warnings, error) {
			values := getMockQueryRangeData(query)
			return values, []string{}, nil
		},
	}
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
		Values: []pModel.SamplePair{{Timestamp: 0, Value: pModel.SampleValue(value)},
			{Timestamp: 0, Value: pModel.SampleValue(value)}},
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
