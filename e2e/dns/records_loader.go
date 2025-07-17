// Package dns ...
package dns

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/public"
)

// RecordsLoader loads DNS records from Route53
type RecordsLoader struct {
	route53Client      *route53.Client
	rhacsZone          *types.HostedZone
	CentralDomainNames []string
	LastResult         []*types.ResourceRecordSet
}

// NewRecordsLoader creates a new instance of RecordsLoader
func NewRecordsLoader(route53Client *route53.Client, central public.CentralRequest) *RecordsLoader {
	rhacsZone, err := getHostedZone(route53Client, central)
	Expect(err).ToNot(HaveOccurred())

	return &RecordsLoader{
		route53Client:      route53Client,
		CentralDomainNames: getCentralDomainNamesSorted(central),
		rhacsZone:          rhacsZone,
	}
}

// LoadDNSRecords loads DNS records from Route53
func (loader *RecordsLoader) LoadDNSRecords() []*types.ResourceRecordSet {
	if len(loader.CentralDomainNames) == 0 {
		return []*types.ResourceRecordSet{}
	}
	idx := 0
	loadingRecords := true
	nextRecord := &loader.CentralDomainNames[idx]
	result := make([]*types.ResourceRecordSet, 0, len(loader.CentralDomainNames))

loading:
	for loadingRecords {
		output, err := loader.route53Client.ListResourceRecordSets(context.Background(), &route53.ListResourceRecordSetsInput{
			HostedZoneId:    loader.rhacsZone.Id,
			StartRecordName: nextRecord,
		})
		Expect(err).ToNot(HaveOccurred())

		for _, recordSet := range output.ResourceRecordSets {
			if *recordSet.Name == loader.CentralDomainNames[idx] {
				result = append(result, &recordSet)
				idx++
				if idx == len(loader.CentralDomainNames) {
					break loading
				}
			}
		}
		loadingRecords = output.IsTruncated
		nextRecord = output.NextRecordName
	}
	loader.LastResult = result
	return result
}

func getHostedZone(route53Client *route53.Client, central public.CentralRequest) (*types.HostedZone, error) {
	hostedZones, err := route53Client.ListHostedZones(context.Background(), &route53.ListHostedZonesInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list hosted zones: %w", err)
	}

	var rhacsZone *types.HostedZone
	for _, zone := range hostedZones.HostedZones {
		// Omit the . at the end of hosted zone name
		name := removeLastChar(*zone.Name)
		if strings.Contains(central.CentralUIURL, name) {
			z := zone
			rhacsZone = &z
			break
		}
	}

	if rhacsZone == nil {
		return nil, fmt.Errorf("failed to find Route53 hosted zone for Central [id: %v, status: %v, UI URL: %v]",
			central.Id, central.Status, central.CentralUIURL)
	}

	return rhacsZone, nil
}

func removeLastChar(s string) string {
	return s[:len(s)-1]
}

func getCentralDomainNamesSorted(central public.CentralRequest) []string {
	uiURL, err := url.Parse(central.CentralUIURL)
	Expect(err).ToNot(HaveOccurred())
	dataHost, _, err := net.SplitHostPort(central.CentralDataURL)
	Expect(err).ToNot(HaveOccurred())

	centralUIDomain := uiURL.Host + "."
	centralDataDomain := dataHost + "."
	domains := []string{centralUIDomain, centralDataDomain}
	sort.Strings(domains)
	return domains
}
