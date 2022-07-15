package presenters

import (
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
)

// ConvertDataPlaneDinosaurStatus ...
func ConvertDataPlaneDinosaurStatus(status map[string]private.DataPlaneCentralStatus) []*dbapi.DataPlaneCentralStatus {
	res := make([]*dbapi.DataPlaneCentralStatus, 0, len(status))

	for k, v := range status {
		c := make([]dbapi.DataPlaneCentralStatusCondition, 0, len(v.Conditions))
		for _, s := range v.Conditions {
			c = append(c, dbapi.DataPlaneCentralStatusCondition{
				Type:    s.Type,
				Reason:  s.Reason,
				Status:  s.Status,
				Message: s.Message,
			})
		}
		res = append(res, &dbapi.DataPlaneCentralStatus{
			CentralClusterID: k,
			Conditions:       c,
			Routes: dbapi.DataPlaneCentralRoutesRequest{
				UIRouter:   v.Routes.UiRouter,
				DataRouter: v.Routes.DataRouter,
			},
			CentralVersion:         v.Versions.Central,
			CentralOperatorVersion: v.Versions.CentralOperator,
		})
	}

	return res
}
