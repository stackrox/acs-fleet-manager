package testutil

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/service/route53"
	. "github.com/onsi/gomega"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
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

// AssertCentralRequestDeleting gets the central requests from the client by ID and asserts the deleting status
func AssertCentralRequestDeleting(ctx context.Context, client *fleetmanager.Client, id string) func() error {
	return AssertCentralRequestStatus(ctx, client, id, constants.CentralRequestStatusDeleting.String())
}

// AssertCentralRequestAccepted gets the central requests from the client by ID and asserts the accepted status
func AssertCentralRequestAccepted(ctx context.Context, client *fleetmanager.Client, id string) func() error {
	return AssertCentralRequestStatus(ctx, client, id, constants.CentralRequestStatusAccepted.String())
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

// AssertReencryptIngressRouteExist asserts that the reencyrpt RouteIngress for a CentralREquest is created
// and stores it in the given route object
func AssertReencryptIngressRouteExist(ctx context.Context, routeService *k8s.RouteService, namespace string, route *openshiftRouteV1.RouteIngress) func() error {
	return func() error {
		reencryptIngress, err := routeService.FindReencryptIngress(ctx, namespace)
		if err != nil {
			return fmt.Errorf("failed finding reencrypt ingress in namespace %s: %v", namespace, err)
		}
		if reencryptIngress == nil {
			return fmt.Errorf("reencrypt ingress in namespace %s not found", namespace)
		}
		*route = *reencryptIngress
		return nil
	}
}
