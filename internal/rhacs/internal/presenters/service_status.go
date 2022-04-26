package presenters

import (
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/api/public"
)

func PresentServiceStatus(userInDenyList bool, dinosaurMaximumCapacityReached bool) *public.ServiceStatus {
	return &public.ServiceStatus{
		Centrals: public.ServiceStatusCentrals{
			MaxCapacityReached: userInDenyList || dinosaurMaximumCapacityReached,
		},
	}
}
