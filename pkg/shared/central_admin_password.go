package shared

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	centralClient "github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/client"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/httputil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO: Should be shared with fleetshard/pkg/central/reconciler.
const (
	revisionAnnotationKey = "rhacs.redhat.com/revision"
)

// EnableAdminPassword enables the admin password for the given central instance.
// This will be done by changing the central CR, and requires appropriate access to K8S.
// The generated password will be returned, and the basic auth provider will be
// NOTE: It's the callers responsibility to reset the admin password afterwards!
func EnableAdminPassword(ctx context.Context, centralID, centralName string, centralUIEndpoint string) (string, error) {
	k8sClient := k8s.CreateClientOrDie()

	// TODO: Needs to be central ID instead of central instance name.
	centralNamespace := fmt.Sprintf("rhacs-%s", centralID)

	// Retrieve existing central.
	central := v1alpha1.Central{}
	err := k8sClient.Get(ctx,
		ctrlClient.ObjectKey{Namespace: centralNamespace, Name: centralName},
		&central,
	)
	if err != nil {
		return "", errors.Wrapf(err, "retrieving central instance %s/%s", centralNamespace, centralName)
	}

	glog.Infof("Found central CR %s in namespace %s", centralName, centralNamespace)

	// If admin password generation disabled is not set, the admin password will be generated, hence no need to update
	// in that case and the default value false.
	if pointer.BoolDeref(central.Spec.Central.AdminPasswordGenerationDisabled, false) {
		// Enable admin password generation, increase the revision, and update the central CR.
		glog.Infof("Setting disable admin password generation to false for central %s/%s",
			centralNamespace, centralName)
		central.Spec.Central.AdminPasswordGenerationDisabled = pointer.Bool(false)
		if err := increaseCentralRevision(&central); err != nil {
			return "", errors.Wrapf(err, "increasing central revision for central instance %s/%s",
				centralNamespace, centralName)
		}
		if err := k8sClient.Update(ctx, &central); err != nil {
			return "", errors.Wrapf(err, "updating central instance %s/%s", centralNamespace, centralName)
		}
		glog.Infof("Updating admin password finished for central %s/%s", centralNamespace, centralName)
	}

	// Wait for the secret to be created with a timeout of 5 minutes, polling in 10 seconds intervals.
	secret, err := waitForSecret(ctx, k8sClient, centralNamespace)
	if err != nil {
		return "", errors.Wrapf(err, "waiting for secret containing admin password for central %s/%s",
			centralNamespace, centralName)
	}

	// Retrieve the "password" key, additionally make sure to trim all spaces from the value.
	password := strings.TrimSpace(string(secret.Data["password"]))
	if password == "" {
		return "", errors.Errorf("admin password was empty for central instance %s/%s",
			centralNamespace, centralName)
	}
	glog.Infof("Retrieve the admin password for central %s/%s", centralNamespace, centralName)

	// Wait for the first successful response from the central API using the basic auth provider.
	if err := waitForBasicAuthProvider(centralUIEndpoint, centralName, centralNamespace, password); err != nil {
		return "", errors.Wrapf(err, "waiting for basic auth provider for central %s/%s",
			centralNamespace, centralName)
	}
	return password, nil
}

// DisableAdminPassword disables the admin password for the given central instance.
// This will be done by changing the central CR, and requires appropriate access to K8S.
func DisableAdminPassword(ctx context.Context, centralID, centralName string) error {
	client := k8s.CreateClientOrDie()

	centralNamespace := fmt.Sprintf("rhacs-%s", centralID)

	// Retrieve existing central.
	central := v1alpha1.Central{}
	err := client.Get(ctx,
		ctrlClient.ObjectKey{Namespace: centralNamespace, Name: centralName},
		&central,
	)
	if err != nil {
		return errors.Wrapf(err, "retrieving central instance %s/%s", centralNamespace, centralName)
	}

	glog.Infof("Found central CR %s in namespace %s", centralName, centralNamespace)

	// If admin password generation disabled is not set, default to true, since we need to explicitly set it in this
	// case to disable it.
	if !pointer.BoolDeref(central.Spec.Central.AdminPasswordGenerationDisabled, false) {
		glog.Infof("Setting disable admin password generation to true for central %s/%s",
			centralNamespace, centralName)
		// Disable admin password generation, increase the revision, and update the central CR.
		central.Spec.Central.AdminPasswordGenerationDisabled = pointer.Bool(true)
		if err := increaseCentralRevision(&central); err != nil {
			return errors.Wrapf(err, "increasing central revision for central instance %s/%s",
				centralNamespace, centralName)
		}
		if err := client.Update(ctx, &central); err != nil {
			return errors.Wrapf(err, "updating central instance %s/%s", centralNamespace, centralName)
		}
		glog.Infof("Updating admin password finished for central %s/%s", centralNamespace, centralName)
	}

	return nil
}

// TODO: Should be shared with fleetshard/pkg/central/reconciler.
func increaseCentralRevision(central *v1alpha1.Central) error {
	revision, err := strconv.Atoi(central.Annotations[revisionAnnotationKey])
	if err != nil {
		return errors.Wrapf(err, "failed to increment central revision %s/%s",
			central.GetNamespace(), central.GetName())
	}
	revision++
	central.Annotations[revisionAnnotationKey] = fmt.Sprintf("%d", revision)
	return nil
}

func waitForSecret(ctx context.Context, client ctrlClient.Client, namespace string) (*corev1.Secret, error) {
	glog.Info("Waiting until secret with admin password is created")
	exists := concurrency.PollWithTimeout(
		func() bool {
			secret := corev1.Secret{} // pragma: allowlist secret
			err := client.Get(ctx, ctrlClient.ObjectKey{Namespace: namespace, Name: "central-htpasswd"}, &secret)
			return err == nil
		}, 10*time.Second, 5*time.Minute)
	if !exists {
		return nil, errors.Errorf(
			"timed out waiting for admin password secret %s/central-htpwass to be created", namespace)
	}

	glog.Infof("Secret with admin password was created successfully")
	secret := corev1.Secret{} // pragma: allowlist secret
	if err := client.Get(ctx, ctrlClient.ObjectKey{Namespace: namespace, Name: "central-htpasswd"}, &secret); err != nil {
		return nil, errors.Wrapf(err, "retrieving secret %s/central-htpasswd", namespace)
	}
	return &secret, nil
}

func waitForBasicAuthProvider(uiEndpoint, name, namespace, password string) error {
	centralClient := centralClient.NewCentralClient(
		private.ManagedCentral{
			Metadata: private.ManagedCentralAllOfMetadata{Name: name, Namespace: namespace},
		}, uiEndpoint, password)

	glog.Infof("Waiting until authentication with basic auth provider works for central %s/%s",
		namespace, name)
	succeeded := concurrency.PollWithTimeout(
		func() bool {
			resp, err := centralClient.SendRequestToCentralRaw(context.Background(), &v1.GetGroupsRequest{},
				http.MethodGet, "/v1/groups")
			if err != nil {
				return false
			}
			return httputil.Is2xxStatusCode(resp.StatusCode)
		},
		10*time.Second, 5*time.Minute)

	if !succeeded {
		return errors.Errorf(
			"no successful request could be done with basic auth provider for central instance %s/%s",
			namespace, name)
	}
	glog.Infof("Authentication with basic auth provider works for central %s/%s", namespace, name)
	return nil
}
