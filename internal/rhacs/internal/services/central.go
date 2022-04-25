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
	List(ctx context.Context, listArgs *services.ListArguments) (dbapi.CentralList, *api.PagingMeta, *errors.ServiceError)
	Get(ctx context.Context, id string) (*dbapi.CentralRequest, *errors.ServiceError)
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
	pagingMeta := &api.PagingMeta{} // FIXME PagingMeta makes no sense for cursor pagination, so it is ignored in centralHandler.List

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
		// FIXME: document behaviour implemented there, this affects the ownership model
	}

	return centralRequestList, pagingMeta, nil
}

// Get returns the Central request that has the specified id.
func (k *centralService) Get(ctx context.Context, id string) (*dbapi.CentralRequest, *errors.ServiceError) {
	return nil, errors.NotFound(fmt.Sprintf("No central found with id %q", id))
}
