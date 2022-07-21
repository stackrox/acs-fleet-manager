package constants

import "github.com/prometheus/client_golang/prometheus"

// DinosaurServerBrokertopicmetricsMessagesInTotalDesc ...
const (
	DinosaurServerBrokertopicmetricsMessagesInTotalDesc            = "Attribute exposed for management (dinosaur.server<type=BrokerTopicMetrics, name=MessagesInPerSec, topic=__dinosaur_operator_canary><>Count)"
	DinosaurServerBrokertopicmetricsBytesInTotalDesc               = "Attribute exposed for management (dinosaur.server<type=BrokerTopicMetrics, name=BytesInPerSec, topic=__dinosaur_operator_canary><>Count)"
	DinosaurServerBrokertopicmetricsBytesOutTotalDesc              = "Attribute exposed for management (dinosaur.server<type=BrokerTopicMetrics, name=BytesOutPerSec, topic=__dinosaur_operator_canary><>Count)"
	DinosaurControllerDinosaurcontrollerOfflinePartitionsCountDesc = "Attribute exposed for management (dinosaur.controller<type=DinosaurController, name=OfflinePartitionsCount><>Value)"
	DinosaurControllerDinosaurcontrollerGlobalPartitionCountDesc   = "Attribute exposed for management (dinosaur.controller<type=DinosaurController, name=GlobalPartitionCount><>Value)"
	DinosaurTopicDinosaurLogLogSizeSumDesc                         = "Attribute exposed for management (dinosaur.log<type=Log, name=Size, topic=__consumer_offsets, partition=18><>Value)"
	DinosaurBrokerQuotaSoftlimitbytesDesc                          = "Attribute exposed for management (io.dinosaur_operator.dinosaur.quotas<type=StorageChecker, name=SoftLimitBytes><>Value)"
	DinosaurBrokerQuotaTotalstorageusedbytesDesc                   = "Attribute exposed for management (io.dinosaur_operator.dinosaur.quotas<type=StorageChecker, name=TotalStorageUsedBytes><>Value)"
	KubeletVolumeStatsAvailableBytesDesc                           = "Number of available bytes in the volume"
	KubeletVolumeStatsUsedBytesDesc                                = "Number of used bytes in the volume"
	HaproxyServerBytesInTotalDesc                                  = "Current total of incoming bytes"
	HaproxyServerBytesOutTotalDesc                                 = "Current total of outgoing bytes"
	DinosaurTopicPartitionsSumDesc                                 = "Number of topic partitions for this Dinosaur"
	DinosaurTopicPartitionsCountDesc                               = "Number of Topics for this Dinosaur"
	DinosaurConsumergroupMembersDesc                               = "Amount of members in a consumer group"
	DinosaurServerSocketServerMetricsConnectionCountDesc           = "Current number of total dinosaur connections"
	DinosaurServerSocketServerMetricsConnectionCreationRateDesc    = "Current rate of connections creation"
)

// MetricsMetadata ...
type MetricsMetadata struct {
	Name           string
	Help           string
	Type           prometheus.ValueType
	TypeName       string
	VariableLabels []string
	ConstantLabels prometheus.Labels
}

// GetMetricsMetaData ...
func GetMetricsMetaData() map[string]MetricsMetadata {
	return map[string]MetricsMetadata{
		"dinosaur_server_brokertopicmetrics_messages_in_total": {
			Name:           "dinosaur_server_brokertopicmetrics_messages_in_total",
			Help:           DinosaurServerBrokertopicmetricsMessagesInTotalDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"topic", "statefulset_kubernetes_io_pod_name", "dinosaur_operator_io_cluster"},
		},
		"dinosaur_server_brokertopicmetrics_bytes_in_total": {
			Name:           "dinosaur_server_brokertopicmetrics_bytes_in_total",
			Help:           DinosaurServerBrokertopicmetricsBytesInTotalDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"topic", "statefulset_kubernetes_io_pod_name", "dinosaur_operator_io_cluster"},
		},
		"dinosaur_server_brokertopicmetrics_bytes_out_total": {
			Name:           "dinosaur_server_brokertopicmetrics_bytes_out_total",
			Help:           DinosaurServerBrokertopicmetricsBytesOutTotalDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"topic", "statefulset_kubernetes_io_pod_name", "dinosaur_operator_io_cluster"},
		},
		"dinosaur_controller_dinosaurcontroller_offline_partitions_count": {
			Name:           "dinosaur_controller_dinosaurcontroller_offline_partitions_count",
			Help:           DinosaurControllerDinosaurcontrollerOfflinePartitionsCountDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"statefulset_kubernetes_io_pod_name", "dinosaur_operator_io_cluster"},
		},
		"dinosaur_controller_dinosaurcontroller_global_partition_count": {
			Name:           "dinosaur_controller_dinosaurcontroller_global_partition_count",
			Help:           DinosaurControllerDinosaurcontrollerGlobalPartitionCountDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"statefulset_kubernetes_io_pod_name", "dinosaur_operator_io_cluster"},
		},
		"dinosaur_topic:dinosaur_log_log_size:sum": {
			Name:           "dinosaur_topic:dinosaur_log_log_size:sum",
			Help:           DinosaurTopicDinosaurLogLogSizeSumDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"topic"},
		},
		"dinosaur_broker_quota_softlimitbytes": {
			Name:           "dinosaur_broker_quota_softlimitbytes",
			Help:           DinosaurBrokerQuotaSoftlimitbytesDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"statefulset_kubernetes_io_pod_name", "dinosaur_operator_io_cluster"},
		},
		"dinosaur_broker_quota_totalstorageusedbytes": {
			Name:           "dinosaur_broker_quota_totalstorageusedbytes",
			Help:           DinosaurBrokerQuotaTotalstorageusedbytesDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"statefulset_kubernetes_io_pod_name", "dinosaur_operator_io_cluster"},
		},
		"kubelet_volume_stats_available_bytes": {
			Name:           "kubelet_volume_stats_available_bytes",
			Help:           KubeletVolumeStatsAvailableBytesDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"persistentvolumeclaim"},
		},
		"kubelet_volume_stats_used_bytes": {
			Name:           "kubelet_volume_stats_used_bytes",
			Help:           KubeletVolumeStatsUsedBytesDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{"persistentvolumeclaim"},
		},
		"dinosaur_namespace:haproxy_server_bytes_in_total:rate5m": {
			Name:           "dinosaur_namespace:haproxy_server_bytes_in_total:rate5m",
			Help:           HaproxyServerBytesInTotalDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{},
		},
		"dinosaur_namespace:haproxy_server_bytes_out_total:rate5m": {
			Name:           "dinosaur_namespace:haproxy_server_bytes_out_total:rate5m",
			Help:           HaproxyServerBytesOutTotalDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{},
		},
		"dinosaur_topic:dinosaur_topic_partitions:sum": {
			Name:           "dinosaur_topic:dinosaur_topic_partitions:sum",
			Help:           DinosaurTopicPartitionsSumDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{},
		},
		"dinosaur_topic:dinosaur_topic_partitions:count": {
			Name:           "dinosaur_topic:dinosaur_topic_partitions:count",
			Help:           DinosaurTopicPartitionsCountDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{},
		},
		"consumergroup:dinosaur_consumergroup_members:count": {
			Name:           "consumergroup:dinosaur_consumergroup_members:count",
			Help:           DinosaurConsumergroupMembersDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{},
		},
		"dinosaur_namespace:dinosaur_server_socket_server_metrics_connection_count:sum": {
			Name:           "dinosaur_namespace:dinosaur_server_socket_server_metrics_connection_count:sum",
			Help:           DinosaurServerSocketServerMetricsConnectionCountDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{},
		},
		"dinosaur_namespace:dinosaur_server_socket_server_metrics_connection_creation_rate:sum": {
			Name:           "dinosaur_namespace:dinosaur_server_socket_server_metrics_connection_creation_rate:sum",
			Help:           DinosaurServerSocketServerMetricsConnectionCreationRateDesc,
			Type:           prometheus.GaugeValue,
			TypeName:       "GAUGE",
			VariableLabels: []string{},
		},
	}
}
