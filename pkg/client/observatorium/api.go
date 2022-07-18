package observatorium

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

type APIObservatoriumService interface {
	GetDinosaurState(name string, namespaceName string) (DinosaurState, error)
	GetMetrics(csMetrics *DinosaurMetrics, resourceNamespace string, rq *MetricsReqParams) error
}
type fetcher struct {
	metric   string
	labels   string
	callback callback
}
type callback func(m Metric)

type ServiceObservatorium struct {
	client *Client
}

func (obs *ServiceObservatorium) GetDinosaurState(name string, resourceNamespace string) (DinosaurState, error) {
	DinosaurState := DinosaurState{}
	c := obs.client
	metric := `dinosaur_operator_resource_state{%s}` // TODO change this to reflect your app specific readness state metric
	labels := fmt.Sprintf(`kind=~'Dinosaur', name=~'%s',resource_namespace=~'%s'`, name, resourceNamespace)
	result := c.Query(metric, labels)
	if result.Err != nil {
		return DinosaurState, result.Err
	}

	for _, s := range result.Vector {

		if s.Value == 1 {
			DinosaurState.State = ClusterStateReady
		} else {
			DinosaurState.State = ClusterStateUnknown
		}
	}
	return DinosaurState, nil
}

func (obs *ServiceObservatorium) GetMetrics(metrics *DinosaurMetrics, namespace string, rq *MetricsReqParams) error {
	failedMetrics := []string{}
	// TODO update metrics names and add more specifics metrics for your service
	fetchers := map[string]fetcher{
		// Check metrics for available disk space per broker
		"kubelet_volume_stats_available_bytes": {
			`kubelet_volume_stats_available_bytes{%s}`,
			fmt.Sprintf(`persistentvolumeclaim=~"data-.*-dinosaur-[0-9]*$", namespace=~'%s'`, namespace),
			func(m Metric) {
				*metrics = append(*metrics, m)
			},
		},
		// Check metrics for used disk space per broker
		"kubelet_volume_stats_used_bytes": {
			`kubelet_volume_stats_used_bytes{%s}`,
			fmt.Sprintf(`persistentvolumeclaim=~"data-.*-dinosaur-[0-9]*$", namespace=~'%s'`, namespace),
			func(m Metric) {
				*metrics = append(*metrics, m)
			},
		},
		// Check metrics for all traffic in/out
		"dinosaur_namespace:haproxy_server_bytes_in_total:rate5m": {
			`dinosaur_namespace:haproxy_server_bytes_in_total:rate5m{%s}`,
			fmt.Sprintf(`exported_namespace=~'%s'`, namespace),
			func(m Metric) {
				*metrics = append(*metrics, m)
			},
		},
		"dinosaur_namespace:haproxy_server_bytes_out_total:rate5m": {
			`dinosaur_namespace:haproxy_server_bytes_out_total:rate5m{%s}`,
			fmt.Sprintf(`exported_namespace=~'%s'`, namespace),
			func(m Metric) {
				*metrics = append(*metrics, m)
			},
		},
		"dinosaur_namespace:dinosaur_server_socket_server_metrics_connection_count:sum": {
			`dinosaur_namespace:dinosaur_server_socket_server_metrics_connection_count:sum{%s}`,
			fmt.Sprintf(`namespace=~'%s'`, namespace),
			func(m Metric) {
				*metrics = append(*metrics, m)
			},
		},
		"dinosaur_namespace:dinosaur_server_socket_server_metrics_connection_creation_rate:sum": {
			`dinosaur_namespace:dinosaur_server_socket_server_metrics_connection_creation_rate:sum{%s}`,
			fmt.Sprintf(`namespace=~'%s'`, namespace),
			func(m Metric) {
				*metrics = append(*metrics, m)
			},
		},
	}

	for msg, f := range fetchers {
		fetchAll := len(rq.Filters) == 0
		if fetchAll {
			result := obs.fetchMetricsResult(rq, &f)
			if result.Err != nil {
				glog.Error("error from metric ", result.Err)
				failedMetrics = append(failedMetrics, fmt.Sprintf("%s: %s", msg, result.Err))
			}
			f.callback(result)
		}
		if !fetchAll {
			for _, filter := range rq.Filters {
				if filter == msg {
					result := obs.fetchMetricsResult(rq, &f)
					if result.Err != nil {
						glog.Error("error from metric ", result.Err)
						failedMetrics = append(failedMetrics, fmt.Sprintf("%s: %s", msg, result.Err))
					}
					f.callback(result)
				}
			}

		}

	}
	if len(failedMetrics) > 0 {
		return errors.New(fmt.Sprintf("Failed to fetch metrics data [%s]", strings.Join(failedMetrics, ",")))
	}
	return nil
}

func (obs *ServiceObservatorium) fetchMetricsResult(rq *MetricsReqParams, f *fetcher) Metric {
	c := obs.client
	var result Metric
	switch rq.ResultType {
	case RangeQuery:
		result = c.QueryRange(f.metric, f.labels, rq.Range)
	case Query:
		result = c.Query(f.metric, f.labels)
	default:
		result = Metric{Err: errors.Errorf("Unsupported Result Type %q", rq.ResultType)}
	}
	return result
}
