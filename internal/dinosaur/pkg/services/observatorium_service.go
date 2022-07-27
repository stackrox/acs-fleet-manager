package services

import (
	"context"
	"fmt"

	"github.com/stackrox/acs-fleet-manager/pkg/client/observatorium"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

var _ ObservatoriumService = &observatoriumService{}

type observatoriumService struct {
	observatorium   *observatorium.Client
	dinosaurService DinosaurService
}

// NewObservatoriumService ...
func NewObservatoriumService(observatorium *observatorium.Client, dinosaurService DinosaurService) ObservatoriumService {
	return &observatoriumService{
		observatorium:   observatorium,
		dinosaurService: dinosaurService,
	}
}

// ObservatoriumService ...
//go:generate moq -out observatorium_service_moq.go . ObservatoriumService
type ObservatoriumService interface {
	GetDinosaurState(name string, namespaceName string) (observatorium.DinosaurState, error)
	GetMetricsByDinosaurID(ctx context.Context, csMetrics *observatorium.DinosaurMetrics, id string, query observatorium.MetricsReqParams) (string, *errors.ServiceError)
}

// GetDinosaurState ...
func (obs observatoriumService) GetDinosaurState(name string, namespaceName string) (observatorium.DinosaurState, error) {
	state, err := obs.observatorium.Service.GetDinosaurState(name, namespaceName)
	if err != nil {
		return state, fmt.Errorf("getting dinosaur state for %q in namespace %q: %w", name, namespaceName, err)
	}
	return state, nil
}

// GetMetricsByDinosaurID ...
func (obs observatoriumService) GetMetricsByDinosaurID(ctx context.Context, dinosaursMetrics *observatorium.DinosaurMetrics, id string, query observatorium.MetricsReqParams) (string, *errors.ServiceError) {
	dinosaurRequest, err := obs.dinosaurService.Get(ctx, id)
	if err != nil {
		return "", err
	}

	getErr := obs.observatorium.Service.GetMetrics(dinosaursMetrics, dinosaurRequest.Namespace, &query)
	if getErr != nil {
		return dinosaurRequest.ID, errors.NewWithCause(errors.ErrorGeneral, getErr, "failed to retrieve metrics")
	}

	return dinosaurRequest.ID, nil
}
