package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/private"
)

func ConvertDataPlaneDinosaurStatus(status map[string]private.DataPlaneDinosaurStatus) []*dbapi.DataPlaneDinosaurStatus {
	// TODO implement converter
	var res []*dbapi.DataPlaneDinosaurStatus
	return res
}
