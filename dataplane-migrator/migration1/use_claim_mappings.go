// Package migration1 ...
package migration1

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"strings"

	apiErrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/pkg/errors"
	centralClientPkg "github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/client"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/operator/pkg/types"
	"github.com/stackrox/rox/pkg/renderer"
	"github.com/stackrox/rox/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
)

// TempFuncName ...
func TempFuncName() {
	// This is needed to make `glog` believe that the flags have already been parsed, otherwise
	// every log messages is prefixed by an error message stating the the flags haven't been
	// parsed.
	_ = flag.CommandLine.Parse([]string{})

	// Always log to stderr by default, required for glog.
	if err := flag.Set("logtostderr", "true"); err != nil {
		glog.Info("Unable to set logtostderr to true")
	}

	glog.Info("Starting application")

	client := k8s.CreateClientOrDie()
	// Algo:
	// 1. List rhacs-{central id} namespaces
	// For every namespace
	//   2. Create central-htpasswd
	//   3. Get Red Hat SSO auth provider <- retry if 401/403
	//   4. Get corresponding groups
	//   5. Delete Red Hat SSO auth provider
	//   6. Delete corresponding groups
	//   7. Re-create Red Hat SSO auth provider with same parameters, but with claim mappings
	//   8. Re-create corresponding groups(modify org_admin group to admin:org:all group)
	nsList := &corev1.NamespaceList{}
	err := client.List(context.Background(), nsList)
	if err != nil {
		glog.Fatal(err)
	}

	for _, ns := range nsList.Items {
		// 26 = 6(prefix + 20(id)
		if strings.HasPrefix(ns.GetName(), "rhacs-") && len(ns.GetName()) == 26 {
			if err := fixCentral(client, ns); err != nil {
				glog.Warning(err)
			}
		}
	}
}

func fixCentral(client ctrlClient.Client, ns corev1.Namespace) error {
	// 2. Create central-htpasswd
	// TODO: random password
	password := "12345"
	htpasswdBytes, err := renderer.CreateHtpasswd(password)
	if err != nil {
		return errors.Wrap(err, "Error generating password")
	}
	data := types.SecretDataMap{
		"htpasswd": htpasswdBytes, // pragma: allowlist secret
	}
	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "central-htpasswd",
			Namespace: ns.GetName(),
			// TODO: correct owner
			OwnerReferences: []metav1.OwnerReference{},
		},
		Data: data,
	}
	if err = client.Create(context.Background(), newSecret); err != nil {
		if !apiErrors.IsAlreadyExists(err) {
			// TODO: should we handle the case when secret still exists
			return errors.Wrap(err, "creating central-htpasswd secret")
		}
	}
	// 3. Get Red Hat SSO auth provider <- retry if 401/403
	central := private.ManagedCentral{
		Metadata: private.ManagedCentralAllOfMetadata{
			Namespace: ns.GetName(),
			Name:      "unknown",
		},
	}
	address, err := getServiceAddress(context.Background(), central, client)
	if err != nil {
		return err
	}
	centralClient := centralClientPkg.NewCentralClient(central, address, password)
	var authProvidersResponse v1.GetAuthProvidersResponse
	err = centralClient.SendRequestToCentral(context.Background(), nil, http.MethodGet, "/v1/authproviders",
		&authProvidersResponse)
	if err != nil {
		// TODO: retry if 401/403
		return errors.Wrap(err, "error sending central request")
	}
	oldAuthProvider := findRHSSOProviderInResponse(authProvidersResponse)
	if oldAuthProvider == nil {
		// TODO: print list of other auth providers
		return errors.New("Red Hat SSO auth provider is not found")
	}

	// 4. Get corresponding groups
	var groupsResponse v1.GetGroupsResponse
	err = centralClient.SendRequestToCentral(context.Background(), &v1.GetGroupsRequest{
		AuthProviderIdOpt: &v1.GetGroupsRequest_AuthProviderId{
			AuthProviderId: oldAuthProvider.GetId(),
		},
	}, http.MethodGet, "/v1/groups",
		&groupsResponse)
	if err != nil {
		return errors.Wrap(err, "error sending central request")
	}
	oldGroups := groupsResponse.GetGroups()
	// 5. Delete Red Hat SSO
	err = centralClient.SendRequestToCentral(context.Background(),
		&v1.DeleteByIDWithForce{
			Id:    oldAuthProvider.GetId(),
			Force: true,
		}, http.MethodDelete, "/v1/authproviders/"+oldAuthProvider.GetId(), &v1.Empty{})
	if err != nil {
		return errors.Wrap(err, "error sending central request")
	}
	// 6. Delete old groups
	for _, oldGroup := range oldGroups {
		id := oldGroup.GetProps().GetId()
		err = centralClient.SendRequestToCentral(context.Background(),
			&v1.DeleteGroupRequest{
				Id:             id,
				Key:            oldGroup.GetProps().GetKey(),
				Value:          oldGroup.GetProps().GetValue(),
				AuthProviderId: oldGroup.GetProps().GetAuthProviderId(),
				Force:          true,
			}, http.MethodDelete, "/v1/groups", &v1.Empty{})
		if err != nil {
			return errors.Wrap(err, "error sending central request")
		}
	}
	// 7. Create new Red Hat SSO auth provider
	newAuthProviderRequest := oldAuthProvider.Clone()
	newAuthProviderRequest.ClaimMappings = map[string]string{
		"realm_access.roles": "groups",
	}
	newAuthProviderRequest.Id = ""
	newAuthProvider, err := centralClient.SendAuthProviderRequest(context.Background(), newAuthProviderRequest)
	if err != nil {
		return errors.Wrap(err, "error sending central request")
	}
	// 8. Re-create groups
	newGroups := make([]*storage.Group, 0, len(oldGroups))
	for _, oldGroup := range oldGroups {
		newValue := utils.IfThenElse(oldGroup.GetProps().GetValue() == "org_admin", "admin:org:all", oldGroup.GetProps().GetValue())
		newGroup := &storage.Group{
			RoleName: oldGroup.GetRoleName(),
			Props: &storage.GroupProperties{
				Traits:         oldGroup.GetProps().GetTraits(),
				AuthProviderId: newAuthProvider.GetId(),
				Key:            oldGroup.GetProps().GetKey(),
				Value:          newValue,
			},
		}
		newGroups = append(newGroups, newGroup)
	}
	for _, newGroup := range newGroups {
		if err = centralClient.SendGroupRequest(context.Background(), newGroup); err != nil {
			return errors.Wrap(err, "error sending central request")
		}
	}
	// 9. Delete central-htpasswd secret
	if err = client.Delete(context.Background(), newSecret); err != nil {
		// TODO: should we handle the case when secret still exists
		return errors.Wrap(err, "error deleting central-htpasswd secret")
	}
	return nil
}

const centralServiceName = "central"

func findRHSSOProviderInResponse(authProvidersResponse v1.GetAuthProvidersResponse) *storage.AuthProvider {
	for _, authProvider := range authProvidersResponse.GetAuthProviders() {
		if authProvider.GetName() == "Red Hat SSO" {
			return authProvider
		}
	}
	return nil
}

// TODO: copied over from fleetshard/.../init_auth.go - perhaps it makes sense to move to a separate pkg
func getServiceAddress(ctx context.Context, central private.ManagedCentral, client ctrlClient.Client) (string, error) {
	service := &corev1.Service{}
	err := client.Get(ctx,
		ctrlClient.ObjectKey{Name: centralServiceName, Namespace: central.Metadata.Namespace},
		service)
	if err != nil {
		return "", errors.Wrapf(err, "getting k8s service for central")
	}
	port, err := getHTTPSServicePort(service)
	if err != nil {
		return "", err
	}
	address := fmt.Sprintf("https://%s.%s.svc.cluster.local:%d", centralServiceName, central.Metadata.Namespace, port)
	return address, nil
}

// TODO: copied over from fleetshard/.../init_auth.go - perhaps it makes sense to move to a separate pkg
func getHTTPSServicePort(service *corev1.Service) (int32, error) {
	for _, servicePort := range service.Spec.Ports {
		if servicePort.Name == "https" {
			return servicePort.Port, nil
		}
	}
	return 0, errors.Errorf("no `https` port is present in %s/%s service", service.Namespace, service.Name)
}
