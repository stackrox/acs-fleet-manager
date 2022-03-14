package observatorium

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"

	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
	"github.com/pkg/errors"
	pAPI "github.com/prometheus/client_golang/api"
	pV1 "github.com/prometheus/client_golang/api/prometheus/v1"
	pModel "github.com/prometheus/common/model"
)

type ClientConfiguration struct {
	BaseURL    string
	Timeout    time.Duration
	Debug      bool
	EnableMock bool
	Insecure   bool
}

type Client struct {
	// Configuration
	Config     *ClientConfiguration
	connection pV1.API
	Service    APIObservatoriumService
}

func NewObservatoriumClient(c *ObservabilityConfiguration) (client *Client, err error) {
	// Create Observatorium client
	observatoriumConfig := &Configuration{
		Timeout:  c.Timeout,
		Debug:    c.Debug,
		Insecure: c.Insecure,
	}

	observatoriumConfig.BaseURL = c.RedHatSsoTokenRefresherUrl

	if c.EnableMock {
		glog.Infof("Using Mock Observatorium Client")
		client, err = NewClientMock(observatoriumConfig)
	} else {
		client, err = NewClient(observatoriumConfig)
	}
	if err != nil {
		glog.Errorf("Unable to create Observatorium client: %s", err)
	}
	return
}

func NewClient(config *Configuration) (*Client, error) {
	client := &Client{
		Config: &ClientConfiguration{
			Timeout:    config.Timeout,
			Debug:      config.Debug,
			EnableMock: false,
			Insecure:   config.Insecure,
		},
	}

	// Ensure baseURL has a trailing slash
	baseURL := strings.TrimSuffix(config.BaseURL, "/")
	client.Config.BaseURL = baseURL + "/"
	client.Service = &ServiceObservatorium{client: client}
	if config.Insecure {
		pAPI.DefaultRoundTripper.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	apiClient, err := pAPI.NewClient(pAPI.Config{
		Address: client.Config.BaseURL,
		RoundTripper: observatoriumRoundTripper{
			config:  *client.Config,
			wrapped: pAPI.DefaultRoundTripper,
		},
	})
	if err != nil {
		return nil, err
	}
	client.connection = pV1.NewAPI(apiClient)
	client.Service = &ServiceObservatorium{client: client}
	return client, nil
}

func NewClientMock(config *Configuration) (*Client, error) {
	client := &Client{
		Config: &ClientConfiguration{
			Timeout:    config.Timeout,
			Debug:      false,
			EnableMock: true,
			Insecure:   config.Insecure,
		},
	}

	client.Service = &ServiceObservatorium{client: client}
	client.connection = client.MockAPI()

	return client, nil
}

type observatoriumRoundTripper struct {
	config  ClientConfiguration
	wrapped http.RoundTripper
}

func (p observatoriumRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	var statusCode int
	path := strings.TrimPrefix(request.URL.String(), p.config.BaseURL)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	start := time.Now()
	resp, err := p.wrapped.RoundTrip(request)
	elapsedTime := time.Since(start)

	if resp != nil {
		statusCode = resp.StatusCode
	}
	metrics.IncreaseObservatoriumRequestCount(statusCode, path, request.Method)
	metrics.UpdateObservatoriumRequestDurationMetric(statusCode, path, request.Method, elapsedTime)

	return resp, err
}

// Send a POST request to server.
func (c *Client) send(query string) (pModel.Value, pV1.Warnings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Config.Timeout)
	defer cancel()
	return c.connection.Query(ctx, query, time.Now())
}

// Send a POST request to server.
func (c *Client) sendRange(query string, bounds pV1.Range) (pModel.Value, pV1.Warnings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Config.Timeout)
	defer cancel()
	return c.connection.QueryRange(ctx, query, bounds)

}

// Query sends a metrics request to server and returns unmashalled Vector response.
// The VectorResult(s) inside will contain either .Value for queries resulting in instant vector,
// or .Values for queries resulting in a range vector.
//
// queryTemplate must contain one %s for labels, e.g. `some_metric{%s}` or `count(some_metric{state="up",%s})`.
// (Undocumented PromQL: empty labels some_metric{} are OK - https://github.com/prometheus/prometheus/issues/3697)
// labels 0 or more constraints separated by comma e.g. `` or `foo="bar",quux="baz"`.
func (c *Client) Query(queryTemplate string, label string) Metric {

	queryString := fmt.Sprintf(queryTemplate, label)
	values, warnings, err := c.send(queryString)

	if len(warnings) > 0 {
		logger.Logger.Warningf("Prometheus client got warnings %s", all(warnings, "and"))
	}
	if err != nil {
		return Metric{Err: err}
	}

	v, ok := values.(pModel.Vector)
	if !ok {
		logger.Logger.Errorf("Prometheus client got data of type %T, but expected model.Vector", values)
		return Metric{Err: errors.Errorf("Prometheus client got data of type %T, but expected model.Vector", values)}
	}
	return Metric{Vector: v}
}
func (c *Client) QueryRange(queryTemplate string, label string, bounds pV1.Range) Metric {

	queryString := fmt.Sprintf(queryTemplate, label)
	values, warnings, err := c.sendRange(queryString, bounds)
	if len(warnings) > 0 {
		logger.Logger.Warningf("Prometheus client got warnings %s", all(warnings, "and"))
	}
	if err != nil {
		return Metric{Err: err}
	}

	m, ok := values.(pModel.Matrix)
	if !ok {
		logger.Logger.Errorf("Prometheus client got data of type %T, but expected model.Matrix", values)
		return Metric{Err: errors.Errorf("Prometheus client got data of type %T, but expected model.Matrix", values)}

	}
	return Metric{Matrix: m}
}

func all(items []string, conjunction string) string {
	count := len(items)
	if count == 0 {
		return ""
	}
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = fmt.Sprintf("'%s'", item)
	}
	if count == 1 {
		return quoted[0]
	}
	head := quoted[0 : count-1]
	tail := quoted[count-1]
	return fmt.Sprintf("%s %s %s", strings.Join(head, ", "), conjunction, tail)
}
