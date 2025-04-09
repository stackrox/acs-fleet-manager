package testutil

import (
	"context"
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go/service/route53"
	. "github.com/onsi/gomega"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
)

// AssertCentralRequestStatus gets the central request from the client by ID and asserts the given status
func AssertCentralRequestStatus(ctx context.Context, client *fleetmanager.Client, id string, status string) func() error {
	return func() error {
		centralRequest, _, err := client.PublicAPI().GetCentralById(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to get central: %w", err)
		}
		if centralRequest.Status != status {
			return fmt.Errorf("expected centralRequest status %s, got %s", status, centralRequest.Status)
		}
		return nil
	}
}

// AssertCentralRequestReady gets the central requests from the client by ID and asserts the ready status
func AssertCentralRequestReady(ctx context.Context, client *fleetmanager.Client, id string) func() error {
	return AssertCentralRequestStatus(ctx, client, id, constants.CentralRequestStatusReady.String())
}

// AssertCentralRequestProvisioning gets the central requests from the client by ID and asserts the provisioning status
func AssertCentralRequestProvisioning(ctx context.Context, client *fleetmanager.Client, id string) func() error {
	return AssertCentralRequestStatus(ctx, client, id, constants.CentralRequestStatusProvisioning.String())
}

// AssertCentralRequestDeprovisioning gets the central requests from the client by ID and asserts the deprovisioning status
func AssertCentralRequestDeprovisioning(ctx context.Context, client *fleetmanager.Client, id string) func() error {
	return AssertCentralRequestStatus(ctx, client, id, constants.CentralRequestStatusDeprovision.String())
}

// AssertDNSMatchesRouter asserts that every domain in centralDomainNames is in recordSets and targets
// the correct hostname given by the routeIngress
func AssertDNSMatchesRouter(centralDomainNames []string, recordSets []*route53.ResourceRecordSet, routeIngress *openshiftRouteV1.RouteIngress) {
	for idx, domain := range centralDomainNames {
		recordSet := recordSets[idx]
		Expect(recordSet.ResourceRecords).To(HaveLen(1))
		record := recordSet.ResourceRecords[0]
		Expect(*recordSet.Name).To(Equal(domain))
		Expect(*record.Value).To(Equal(routeIngress.RouterCanonicalHostname)) // TODO use route specific ingress instead of comparing with reencryptIngress for all cases
	}
}

// AssertReencryptIngressRouteExist asserts that the reencrypt RouteIngress for a CentralRequest is created
// and stores it in the given RouteIngress object
func AssertReencryptIngressRouteExist(ctx context.Context, routeService *k8s.RouteService, centralRequest public.CentralRequest, ingress *openshiftRouteV1.RouteIngress) func(g Gomega) {
	namespace, err := services.FormatNamespace(centralRequest.Id)
	Expect(err).ToNot(HaveOccurred())
	centralUIURL, err := url.Parse(centralRequest.CentralUIURL)
	Expect(err).ToNot(HaveOccurred())
	return func(g Gomega) {
		ingresses, err := routeService.FindAdmittedIngresses(ctx, namespace)
		g.Expect(err).ToNot(HaveOccurred(), "failed to find reencrypt ingresses in namespace %s", namespace)
		g.Expect(ingresses).To(ContainElement(WithTransform(getRouteIngressHost, Equal(centralUIURL.Host)), &ingress))
	}
}

func getRouteIngressHost(ingress openshiftRouteV1.RouteIngress) string {
	return ingress.Host
}
