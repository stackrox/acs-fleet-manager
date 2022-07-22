package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/metrics"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"

	"github.com/gorilla/mux"
	"github.com/stackrox/acs-fleet-manager/pkg/client/observatorium"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

type metricsHandler struct {
	service services.ObservatoriumService
}

// NewMetricsHandler ...
func NewMetricsHandler(service services.ObservatoriumService) *metricsHandler {
	return &metricsHandler{
		service: service,
	}
}

// FederateMetrics ...
func (h metricsHandler) FederateMetrics(w http.ResponseWriter, r *http.Request) {
	dinosaurID := strings.TrimSpace(mux.Vars(r)["id"])
	if dinosaurID == "" {
		shared.HandleError(r, w, &errors.ServiceError{
			Code:     errors.ErrorBadRequest,
			Reason:   "missing path parameter: dinosaur id",
			HTTPCode: http.StatusBadRequest,
		})
		return
	}

	dinosaurMetrics := &observatorium.DinosaurMetrics{}
	params := observatorium.MetricsReqParams{
		ResultType: observatorium.Query,
	}

	_, err := h.service.GetMetricsByDinosaurID(r.Context(), dinosaurMetrics, dinosaurID, params)
	if err != nil {
		if err.Code == errors.ErrorNotFound {
			shared.HandleError(r, w, err)
		} else {
			glog.Errorf("error getting metrics: %v", err)
			sentry.CaptureException(err)
			shared.HandleError(r, w, &errors.ServiceError{
				Code:     err.Code,
				Reason:   "error getting metrics",
				HTTPCode: http.StatusInternalServerError,
			})
		}
		return
	}

	// Define metric collector
	collector := metrics.NewFederatedUserMetricsCollector(dinosaurMetrics)
	registry := prometheus.NewPedanticRegistry()
	registry.MustRegister(collector)

	promHandler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		ErrorHandling: promhttp.HTTPErrorOnError,
	})
	promHandler.ServeHTTP(w, r)
}

// GetMetricsByRangeQuery ...
func (h metricsHandler) GetMetricsByRangeQuery(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	params := observatorium.MetricsReqParams{}
	query := r.URL.Query()
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidatQueryParam(query, "duration"),
			handlers.ValidatQueryParam(query, "interval"),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			ctx := r.Context()
			params.ResultType = observatorium.RangeQuery
			extractMetricsQueryParams(r, &params)
			dinosaurMetrics := &observatorium.DinosaurMetrics{}
			foundDinosaurID, err := h.service.GetMetricsByDinosaurID(ctx, dinosaurMetrics, id, params)
			if err != nil {
				return nil, err
			}
			metricList := public.MetricsRangeQueryList{
				Kind: "MetricsRangeQueryList",
				Id:   foundDinosaurID,
			}
			metricsResult, err := presenters.PresentMetricsByRangeQuery(dinosaurMetrics)
			if err != nil {
				return nil, err
			}
			metricList.Items = metricsResult

			return metricList, nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

// GetMetricsByInstantQuery ...
func (h metricsHandler) GetMetricsByInstantQuery(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	params := observatorium.MetricsReqParams{}
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			ctx := r.Context()
			params.ResultType = observatorium.Query
			extractMetricsQueryParams(r, &params)
			dinosaurMetrics := &observatorium.DinosaurMetrics{}
			foundDinosaurID, err := h.service.GetMetricsByDinosaurID(ctx, dinosaurMetrics, id, params)
			if err != nil {
				return nil, err
			}
			metricList := public.MetricsInstantQueryList{
				Kind: "MetricsInstantQueryList",
				Id:   foundDinosaurID,
			}
			metricsResult, err := presenters.PresentMetricsByInstantQuery(dinosaurMetrics)
			if err != nil {
				return nil, err
			}
			metricList.Items = metricsResult

			return metricList, nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

func extractMetricsQueryParams(r *http.Request, q *observatorium.MetricsReqParams) {
	q.FillDefaults()
	queryParams := r.URL.Query()
	if dur := queryParams.Get("duration"); dur != "" {
		if num, err := strconv.ParseInt(dur, 10, 64); err == nil {
			duration := time.Duration(num) * time.Minute
			q.Start = q.End.Add(-duration)
		}
	}
	if step := queryParams.Get("interval"); step != "" {
		if num, err := strconv.Atoi(step); err == nil {
			q.Step = time.Duration(num) * time.Second
		}
	}
	if filters, ok := queryParams["filters"]; ok && len(filters) > 0 {
		q.Filters = filters
	}

}
