package services

import (
	"context"
	"fmt"

	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
)

// A CentralService exposes methods to retrieve, manipulate and store Central requests.
//go:generate moq -out centralservice_moq.go . CentralService
type CentralService interface {
	// TODO following internal/dinosaur/internal/services/dinosaur.go
	HasAvailableCapacity() (bool, *errors.ServiceError)
	List(ctx context.Context, listArgs *services.ListArguments) (dbapi.CentralList, *api.PagingMeta, *errors.ServiceError)
	Get(ctx context.Context, id string) (*dbapi.CentralRequest, *errors.ServiceError)
	Update(centralRequest *dbapi.CentralRequest) *errors.ServiceError
}

var _ CentralService = &centralService{}

type centralService struct {}

func NewCentralService() *centralService {
	return &centralService{}
}

// List returns all Central requests that are owned by the organisation of the user authenticated for the request.
func (k *centralService) List(ctx context.Context, listArgs *services.ListArguments) (dbapi.CentralList, *api.PagingMeta, *errors.ServiceError) {
	// FIXME
	var centralRequestList dbapi.CentralList
	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
	}

	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, nil, errors.NewWithCause(errors.ErrorUnauthenticated, err, "user not authenticated")
	}

	
	if !auth.GetIsAdminFromContext(ctx) {
		user := auth.GetUsernameFromClaims(claims)
		if user == "" {
			return nil, nil, errors.Unauthenticated("user not authenticated")
		}

		orgId := auth.GetOrgIdFromClaims(claims)
		logger.Logger.Warningf("Request identified with user=%q and organisation=%q", user, orgId)
		// TODO filter by organisation id, see `func (k *dinosaurService) List` at internal/dinosaur/internal/services/dinosaur.go 
		// FIXME: document behaviour implemented there, this affects the ownership model, therefore the API definition
	}

	return centralRequestList, pagingMeta, nil
}

// Get returns the Central request that has the specified id.
func (k *centralService) Get(ctx context.Context, id string) (*dbapi.CentralRequest, *errors.ServiceError) {
	return nil, errors.NotFound(fmt.Sprintf("No central found with id %q", id))
}

func (k *centralService) HasAvailableCapacity() (bool, *errors.ServiceError) {
	// FIXME
	return true, nil
}

func (k *centralService) Update(centralRequest *dbapi.CentralRequest) *errors.ServiceError {
	// TODO
	// FIXME: how can this work without taking a ctx?
	return nil
}