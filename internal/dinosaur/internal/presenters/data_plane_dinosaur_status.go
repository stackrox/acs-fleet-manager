package presenters

import (
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/api/private"
)

func ConvertDataPlaneDinosaurStatus(status map[string]private.DataPlaneDinosaurStatus) []*dbapi.DataPlaneDinosaurStatus {
	// TODO implement converter
	var res []*dbapi.DataPlaneDinosaurStatus
	return res
}
