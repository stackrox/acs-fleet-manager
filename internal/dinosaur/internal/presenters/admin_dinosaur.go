package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services/account"
)

func PresentDinosaurRequestAdminEndpoint(dinosaurRequest *dbapi.DinosaurRequest, accountService account.AccountService) (*private.Dinosaur, *errors.ServiceError) {
	// TODO implement presenter
	var res *private.Dinosaur
	return res, nil
}
