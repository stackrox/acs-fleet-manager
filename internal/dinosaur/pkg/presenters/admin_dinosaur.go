package presenters

import (
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/fleet-manager-pkg/pkg/services/account"
)

// PresentDinosaurRequestAdminEndpoint ...
func PresentDinosaurRequestAdminEndpoint(dinosaurRequest *dbapi.CentralRequest, accountService account.AccountService) (*private.Dinosaur, *errors.ServiceError) {
	// TODO implement presenter
	var res *private.Dinosaur
	return res, nil
}
