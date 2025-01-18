package dns

import (
	"github.com/aws/aws-sdk-go/service/route53"
	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
)

// CleanupCentralRequestRecords deletes all route53 resoruces associated with the centralRequest
func CleanupCentralRequestRecords(route53Client *route53.Route53, centralRequest public.CentralRequest) {
	dnsLoader := NewRecordsLoader(route53Client, centralRequest)
	recordSets := dnsLoader.LoadDNSRecords()

	action := string(services.CentralRoutesActionDelete)
	changes := []*route53.Change{}
	for _, rs := range recordSets {
		c := &route53.Change{
			Action:            &action,
			ResourceRecordSet: rs,
		}
		changes = append(changes, c)
	}

	if len(changes) == 0 {
		return
	}

	_, err := route53Client.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
		HostedZoneId: dnsLoader.rhacsZone.Name,
		ChangeBatch:  &route53.ChangeBatch{Changes: changes},
	})

	Expect(err).ToNot(HaveOccurred())
}
