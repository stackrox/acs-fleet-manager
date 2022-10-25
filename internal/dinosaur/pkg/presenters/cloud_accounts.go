package presenters

import (
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
)

// PresentCloudAccounts ...
func PresentCloudAccounts(cloudAccounts []*amsv1.CloudAccount) public.CloudAccountsList {
	transformedCloudAccounts := make([]public.CloudAccount, 0, len(cloudAccounts))
	for _, cloudAccount := range cloudAccounts {
		transformedCloudAccounts = append(transformedCloudAccounts, public.CloudAccount{
			CloudAccountId:  cloudAccount.CloudAccountID(),
			CloudProviderId: cloudAccount.CloudProviderID(),
		})
	}
	return public.CloudAccountsList{
		CloudAccounts: transformedCloudAccounts,
	}
}
