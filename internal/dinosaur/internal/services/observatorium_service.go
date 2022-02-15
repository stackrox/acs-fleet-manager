package services

import (
	"context"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
)

var _ ObservatoriumService = &observatoriumService{}

type observatoriumService struct {
	observatorium   *observatorium.Client
	dinosaurService DinosaurService
}

func NewObservatoriumService(observatorium *observatorium.Client, dinosaurService DinosaurService) ObservatoriumService {
	return &observatoriumService{
		observatorium:   observatorium,
		dinosaurService: dinosaurService,
	}
}

//go:generate moq -out observatorium_service_moq.go . ObservatoriumService
type ObservatoriumService interface {
	GetDinosaurState(name string, namespaceName string) (observatorium.DinosaurState, error)
	GetMetricsByDinosaurId(ctx context.Context, csMetrics *observatorium.DinosaurMetrics, id string, query observatorium.MetricsReqParams) (string, *errors.ServiceError)
}

func (obs observatoriumService) GetDinosaurState(name string, namespaceName string) (observatorium.DinosaurState, error) {
	return obs.observatorium.Service.GetDinosaurState(name, namespaceName)
}

func (obs observatoriumService) GetMetricsByDinosaurId(ctx context.Context, dinosaursMetrics *observatorium.DinosaurMetrics, id string, query observatorium.MetricsReqParams) (string, *errors.ServiceError) {
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
