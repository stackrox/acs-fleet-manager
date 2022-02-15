package observatorium

import (
	prom_v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	pModel "github.com/prometheus/common/model"

	"time"
)

type State string
type ResultType string

const (
	ClusterStateUnknown State      = "unknown"
	ClusterStateReady   State      = "ready"
	RangeQuery          ResultType = "query_range"
	Query               ResultType = "query"
)

type DinosaurState struct {
	State State `json:",omitempty"`
}

type DinosaurMetrics []Metric

// Metric holds the Prometheus Matrix or Vector model, which contains instant vector or range vector with time series (depending on result type)
type Metric struct {
	Matrix pModel.Matrix `json:"matrix"`
	Vector pModel.Vector `json:"vector"`
	Err    error         `json:"-"`
}

// MetricsReqParams holds common parameters for all kinds of range queries and instant quries
type MetricsReqParams struct {
	Filters    []string
	ResultType ResultType
	prom_v1.Range
}

// FillDefaults fills the struct with default parameters
func (q *MetricsReqParams) FillDefaults() {
	q.End = time.Now()
	q.Start = q.End.Add(-5 * time.Minute)
	q.Step = 30 * time.Second

}
