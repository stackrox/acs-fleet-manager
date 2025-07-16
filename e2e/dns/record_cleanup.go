package dns

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
)

// CleanupCentralRequestRecords deletes all route53 resoruces associated with the centralRequest
func CleanupCentralRequestRecords(route53Client *route53.Client, centralRequest public.CentralRequest) {
	dnsLoader := NewRecordsLoader(route53Client, centralRequest)
	recordSets := dnsLoader.LoadDNSRecords()

	action, err := services.CentralRoutesActionToRoute53ChangeAction(services.CentralRoutesActionDelete)
	Expect(err).ToNot(HaveOccurred())

	changes := []types.Change{}
	for _, rs := range recordSets {
		c := types.Change{
			Action:            action,
			ResourceRecordSet: rs,
		}
		changes = append(changes, c)
	}

	if len(changes) == 0 {
		return
	}

	_, err = route53Client.ChangeResourceRecordSets(context.Background(), &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: dnsLoader.rhacsZone.Name,
		ChangeBatch:  &types.ChangeBatch{Changes: changes},
	})

	Expect(err).ToNot(HaveOccurred())
}
