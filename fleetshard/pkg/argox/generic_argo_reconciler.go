package argox

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	argocd "github.com/stackrox/acs-fleet-manager/pkg/argocd/apis/application/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/json"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcileApplications is a generic argocd reconciler function that manages a set of argoCd applications.
// This is useful, because it knows how to add/update/delete applications.
// The selector is important, as it is used to build the current state of the applications.
// !!! Important: If an application matching the selector is present on the cluster, but not in the desired state, it will be deleted
// So make sure that the selector is unique to the applications you want to manage with this function.
func ReconcileApplications(
	ctx context.Context,
	client ctrlClient.Client,
	namespace string,
	existingStateSelector map[string]string,
	applications []*argocd.Application,
) error {

	if len(existingStateSelector) == 0 {
		return fmt.Errorf("existingStateSelector must not be empty")
	}

	glog.V(10).Infof("reconciling %d argocd applications in namespace %q", len(applications), namespace)
	for _, desiredApplication := range applications {

		jsonBytes, err := getJsonString(desiredApplication)
		if err != nil {
			return err
		}

		if desiredApplication.Labels == nil {
			desiredApplication.Labels = make(map[string]string)
		}

		// Ensuring the desired applications have labels matching the existingStateSelector
		for k, v := range existingStateSelector {
			desiredApplication.Labels[k] = v
		}

		// Ensuring the namespace is set
		desiredApplication.Namespace = namespace

		if err := reconcile(ctx, client, desiredApplication, string(jsonBytes)); err != nil {
			return err
		}
	}

	existingApplications := argocd.ApplicationList{}
	err := client.List(ctx, &existingApplications, ctrlClient.InNamespace(namespace), ctrlClient.MatchingLabels(existingStateSelector))
	if err != nil {
		return err
	}

	existingApplicationsMap := map[string]argocd.Application{}
	for i := range existingApplications.Items {
		existingApplicationsMap[existingApplications.Items[i].Name] = existingApplications.Items[i]
	}

	desiredApplicationsNames := make(map[string]struct{})
	for _, app := range applications {
		desiredApplicationsNames[app.Name] = struct{}{}
	}

	for existingApplicationName, existingApplication := range existingApplicationsMap {
		_, isInDesiredState := desiredApplicationsNames[existingApplicationName]
		if isInDesiredState {
			glog.V(10).Infof("argocd application %q is in the desired state, skipping deletion", existingApplicationName)
			continue
		}
		// The application does not exist in the desired state, delete it
		glog.V(10).Infof("deleting argocd application %q because it is not in the desired state", existingApplicationName)
		err := client.Delete(ctx, &existingApplication)
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func ReconcileApplication(
	ctx context.Context,
	client ctrlClient.Client,
	desiredApplication *argocd.Application,
) error {
	if len(desiredApplication.Name) == 0 {
		return fmt.Errorf("desiredApplication.Name must not be empty")
	}
	if len(desiredApplication.Namespace) == 0 {
		return fmt.Errorf("desiredApplication.Namespace must not be empty")
	}
	desiredConfigString, err := getJsonString(desiredApplication)
	if err != nil {
		return err
	}
	return reconcile(ctx, client, desiredApplication, desiredConfigString)
}

func reconcile(
	ctx context.Context,
	client ctrlClient.Client,
	desiredApplication *argocd.Application,
	desiredConfigString string,
) error {

	glog.V(10).Infof("reconciling argocd application %q", desiredApplication.Name)

	if desiredApplication.Annotations == nil {
		desiredApplication.Annotations = make(map[string]string)
	}

	// Setting the last-applied-configuration annotation for change detection
	const lastAppliedConfigurationAnnotation = "rhacs.redhat.com/last-applied-configuration"
	desiredApplication.Annotations[lastAppliedConfigurationAnnotation] = desiredConfigString

	// ------------------------------------
	// Creating or updating the application
	// ------------------------------------

	existingApplication := argocd.Application{}
	err := client.Get(ctx, ctrlClient.ObjectKeyFromObject(desiredApplication), &existingApplication)
	if errors.IsNotFound(err) {
		// The application does not exist, create it
		glog.V(10).Infof("argocd application %q does not exist, creating it", desiredApplication.Name)
		return client.Create(ctx, desiredApplication)
	} else if err != nil {
		return err
	}

	// The application exists, check if it needs to be updated
	lastAppliedConfig, hasLastAppliedConfig := existingApplication.Annotations[lastAppliedConfigurationAnnotation]
	needsUpdate := !hasLastAppliedConfig || lastAppliedConfig != desiredConfigString

	if needsUpdate {
		// Update the application
		glog.V(10).Infof("updating argocd application %q because changes were detected", desiredApplication.Name)
		desiredApplication.SetResourceVersion(existingApplication.GetResourceVersion())
		if err := client.Update(ctx, desiredApplication); err != nil {
			return err
		}
	} else {
		glog.V(10).Infof("argocd application %q is up to date, skipping update", desiredApplication.Name)
	}

	return nil
}

func getJsonString(obj interface{}) (string, error) {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}
