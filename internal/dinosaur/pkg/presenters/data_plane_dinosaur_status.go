package presenters

import (
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
)

func ConvertDataPlaneDinosaurStatus(status map[string]private.DataPlaneDinosaurStatus) []*dbapi.DataPlaneDinosaurStatus {
	res := make([]*dbapi.DataPlaneDinosaurStatus, 0, len(status))

	for k, v := range status {
		c := make([]dbapi.DataPlaneDinosaurStatusCondition, 0, len(v.Conditions))
		var routes []dbapi.DataPlaneDinosaurRouteRequest
		for _, s := range v.Conditions {
			c = append(c, dbapi.DataPlaneDinosaurStatusCondition{
				Type:    s.Type,
				Reason:  s.Reason,
				Status:  s.Status,
				Message: s.Message,
			})
		}
		if v.Routes != nil {
			routes = make([]dbapi.DataPlaneDinosaurRouteRequest, 0, len(*v.Routes))
			for _, ro := range *v.Routes {
				routes = append(routes, dbapi.DataPlaneDinosaurRouteRequest{
					Name:   ro.Name,
					Prefix: ro.Prefix,
					Router: ro.Router,
				})
			}
		}
		res = append(res, &dbapi.DataPlaneDinosaurStatus{
			DinosaurClusterId:       k,
			Conditions:              c,
			Routes:                  routes,
			DinosaurVersion:         v.Versions.Dinosaur,
			DinosaurOperatorVersion: v.Versions.DinosaurOperator,
		})
	}

	return res
}
