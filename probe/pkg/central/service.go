// Package central is responsible for operations with central
package central

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/antihax/optional"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager/impl"
	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/utils"
)

// ErrNotFound indicates that given central is not found
var ErrNotFound = errors.New("central not found")

// Service provides basic operations with Centrals
//
//go:generate moq -rm -out service_moq.go . Service
type Service interface {
	Get(ctx context.Context, id string) (public.CentralRequest, error)
	List(ctx context.Context, spec Spec) ([]public.CentralRequest, error)
	ListSpecs(ctx context.Context) ([]Spec, error)
	Delete(ctx context.Context, id string) error
	Create(ctx context.Context, name string, spec Spec) (public.CentralRequest, error)
	Ping(ctx context.Context, url string) error
}

// NewService creates a new central service.
// this function also checks that serviceImpl implements Service
func NewService(ctx context.Context, config config.Config) (Service, error) {
	auth, err := impl.NewAuth(ctx, config.AuthType, impl.OptionFromEnv())
	if err != nil {
		return nil, errors.Wrap(err, "failed to create fleet manager authentication")
	}

	client, err := impl.NewClient(config.FleetManagerEndpoint, auth, impl.WithUserAgent("fleet-manager-probe-service"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create fleet manager client")
	}

	return &serviceImpl{
		fleetManagerPublicAPI: client.PublicAPI(),
		httpClient:            &http.Client{Timeout: config.ProbeHTTPRequestTimeout},
	}, nil
}

type serviceImpl struct {
	fleetManagerPublicAPI fleetmanager.PublicAPI
	httpClient            *http.Client
}

// Get gets central by given ID
func (s *serviceImpl) Get(ctx context.Context, id string) (public.CentralRequest, error) {
	centralResp, response, err := s.fleetManagerPublicAPI.GetCentralById(ctx, id)
	defer utils.IgnoreError(closeBodyIfNonEmpty(response))
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			return public.CentralRequest{}, ErrNotFound
		}
		err = errors.WithMessage(err, extractCentralError(response))
		err = errors.Wrapf(err, "central instance %s not reachable", id)
		glog.Error(err)
		return public.CentralRequest{}, err
	}
	return centralResp, nil
}

// List lists central with a given filter
func (s *serviceImpl) List(ctx context.Context, spec Spec) ([]public.CentralRequest, error) {
	centralList, resp, err := s.fleetManagerPublicAPI.GetCentrals(ctx, &public.GetCentralsOpts{
		Search: optional.NewString(spec.Query()),
	})
	defer utils.IgnoreError(closeBodyIfNonEmpty(resp))
	if err != nil {
		err = errors.WithMessage(err, extractCentralError(resp))
		err = errors.Wrap(err, "could not list centrals")
		glog.Error(err)
		return []public.CentralRequest{}, err
	}
	return centralList.Items, nil
}

func (s *serviceImpl) ListSpecs(ctx context.Context) ([]Spec, error) {
	cloudProviders, response, err := s.fleetManagerPublicAPI.GetCloudProviders(ctx, nil)
	defer utils.IgnoreError(closeBodyIfNonEmpty(response))
	if err != nil {
		err = errors.WithMessage(err, extractCentralError(response))
		err = errors.Wrap(err, "could not list cloud providers")
		glog.Error(err)
		return []Spec{}, err
	}
	var specs []Spec
	for _, cloudProvider := range cloudProviders.Items {
		specs = s.appendRegions(ctx, cloudProvider, specs)
	}

	return specs, nil
}

func (s *serviceImpl) appendRegions(ctx context.Context, cloudProvider public.CloudProvider, specs []Spec) []Spec {
	regions, response, err := s.fleetManagerPublicAPI.GetCloudProviderRegions(ctx, cloudProvider.Id, nil)
	defer utils.IgnoreError(closeBodyIfNonEmpty(response))
	if err != nil {
		glog.Errorf("unable to get regions for the cloud provider %s: %v", cloudProvider.Id, err)
		return specs
	}
	for _, region := range regions.Items {
		specs = append(specs, Spec{CloudProvider: cloudProvider.Id, Region: region.Id})
	}
	return specs
}

// Delete calls Fleet Manager to delete the central instance with the given ID.
func (s *serviceImpl) Delete(ctx context.Context, id string) error {
	resp, err := s.fleetManagerPublicAPI.DeleteCentralById(ctx, id, true)
	glog.Infof("deletion of central instance %s requested", id)
	defer utils.IgnoreError(closeBodyIfNonEmpty(resp))
	if err != nil {
		err = errors.WithMessage(err, extractCentralError(resp))
		return errors.Wrapf(err, "deletion of central instance %s failed", id)
	}
	return nil
}

// Create creates new central
func (s *serviceImpl) Create(ctx context.Context, name string, spec Spec) (public.CentralRequest, error) {
	request := public.CentralRequestPayload{
		Name:          name,
		MultiAz:       true,
		CloudProvider: spec.CloudProvider,
		Region:        spec.Region,
	}
	central, resp, err := s.fleetManagerPublicAPI.CreateCentral(ctx, true, request)
	defer utils.IgnoreError(closeBodyIfNonEmpty(resp))
	if central.Id == "" {
		glog.Info("creation of central instance requested - got empty response")
	} else {
		glog.Infof("creation of central instance %s requested", central.Id)
	}
	if err != nil {
		err = errors.WithMessage(err, extractCentralError(resp))
		return public.CentralRequest{}, errors.Wrap(err, "creation of central instance failed")
	}
	return central, nil
}

// Ping checks if the given central endpoint is available
func (s *serviceImpl) Ping(ctx context.Context, url string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		err = errors.Wrap(err, "failed to create request")
		glog.Error(err)
		return err
	}
	response, err := s.httpClient.Do(request)
	defer utils.IgnoreError(closeBodyIfNonEmpty(response))
	if err != nil {
		err = errors.Wrap(err, "URL not reachable")
		glog.Error(err)
		return err
	}
	if !httputil.Is2xxStatusCode(response.StatusCode) {
		err = errors.Errorf("URL ping did not succeed: %s", extractCentralError(response))
		glog.Warning(err)
		return err
	}
	return nil
}

func closeBodyIfNonEmpty(resp *http.Response) func() error {
	if resp == nil || resp.Body == nil {
		return func() error {
			return nil
		}
	}
	return func() error {
		return errors.Wrap(resp.Body.Close(), "closing response body")
	}
}

func extractCentralError(resp *http.Response) string {
	var centralError public.Error
	if resp == nil || resp.Body == nil {
		return ""
	}
	if err := json.NewDecoder(resp.Body).Decode(&centralError); err != nil {
		return "parsing HTTP response"
	}
	return fmt.Sprintf("request responded with %d: central error %s and reason %s", resp.StatusCode,
		centralError.Code, centralError.Reason)
}

// Spec the desired central specification
type Spec struct {
	CloudProvider string
	Region        string
}

// Query returns search query for call to the Fleet Manager API
func (s Spec) Query() string {
	return fmt.Sprintf("region = %s and cloud_provider = %s", s.Region, s.CloudProvider)
}
