package dns

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/service/route53"
	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
)

// RecordsLoader loads DNS records from Route53
type RecordsLoader struct {
	route53Client      *route53.Route53
	rhacsZone          *route53.HostedZone
	CentralDomainNames []string
	LastResult         []*route53.ResourceRecordSet
}

// NewRecordsLoader creates a new instance of RecordsLoader
func NewRecordsLoader(route53Client *route53.Route53, central *public.CentralRequest) *RecordsLoader {
	rhacsZone, err := getHostedZone(route53Client, central)
	Expect(err).ToNot(HaveOccurred())

	return &RecordsLoader{
		route53Client:      route53Client,
		CentralDomainNames: getCentralDomainNamesSorted(central),
		rhacsZone:          rhacsZone,
	}
}

// LoadDNSRecords loads DNS records from Route53
func (loader *RecordsLoader) LoadDNSRecords() []*route53.ResourceRecordSet {
	if len(loader.CentralDomainNames) == 0 {
		return []*route53.ResourceRecordSet{}
	}
	idx := 0
	loadingRecords := true
	nextRecord := &loader.CentralDomainNames[idx]
	result := make([]*route53.ResourceRecordSet, 0, len(loader.CentralDomainNames))

loading:
	for loadingRecords {
		output, err := loader.route53Client.ListResourceRecordSets(&route53.ListResourceRecordSetsInput{
			HostedZoneId:    loader.rhacsZone.Id,
			StartRecordName: nextRecord,
		})
		Expect(err).ToNot(HaveOccurred())

		for _, recordSet := range output.ResourceRecordSets {
			if *recordSet.Name == loader.CentralDomainNames[idx] {
				result = append(result, recordSet)
				idx++
				if idx == len(loader.CentralDomainNames) {
					break loading
				}
			}
		}
		loadingRecords = *output.IsTruncated
		nextRecord = output.NextRecordName
	}
	loader.LastResult = result
	return result
}

func getHostedZone(route53Client *route53.Route53, central *public.CentralRequest) (*route53.HostedZone, error) {
	hostedZones, err := route53Client.ListHostedZones(&route53.ListHostedZonesInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list hosted zones: %w", err)
	}

	var rhacsZone *route53.HostedZone
	for _, zone := range hostedZones.HostedZones {
		// Omit the . at the end of hosted zone name
		name := removeLastChar(*zone.Name)
		if strings.Contains(central.CentralUIURL, name) {
			rhacsZone = zone
			break
		}
	}

	if rhacsZone == nil {
		return nil, fmt.Errorf("failed to find Route53 hosted zone for Central UI URL %v", central.CentralUIURL)
	}

	return rhacsZone, nil
}

func removeLastChar(s string) string {
	return s[:len(s)-1]
}

func getCentralDomainNamesSorted(central *public.CentralRequest) []string {
	uiURL, err := url.Parse(central.CentralUIURL)
	Expect(err).ToNot(HaveOccurred())
	dataURL, err := url.Parse(central.CentralDataURL)
	Expect(err).ToNot(HaveOccurred())

	centralUIDomain := uiURL.Host + "."
	centralDataDomain := dataURL.Host + "."
	domains := []string{centralUIDomain, centralDataDomain}
	sort.Strings(domains)
	return domains
}
