package shared

import (
	"context"
	"fmt"
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
	"net/http"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
	"time"
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

	// If admin password generation disabled is not set, the admin password will be generated, hence no need to update
	// in that case and the default value false.
	if pointer.BoolDeref(central.Spec.Central.AdminPasswordGenerationDisabled, false) {
		// Enable admin password generation, increase the revision, and update the central CR.
		central.Spec.Central.AdminPasswordGenerationDisabled = pointer.Bool(false)
		if err := increaseCentralRevision(&central); err != nil {
			return "", errors.Wrapf(err, "increasing central revision for central instance %s/%s",
				centralNamespace, centralName)
		}
		if err := k8sClient.Update(ctx, &central); err != nil {
			return "", errors.Wrapf(err, "updating central instance %s/%s", centralNamespace, centralName)
		}
	}

	// Wait for the secret to be created with a timeout of 5 minutes, polling in 10 seconds intervals.
	exists := concurrency.PollWithTimeout(
		func() bool {
			secret := corev1.Secret{} // pragma: allowlist secret
			err := k8sClient.Get(ctx, ctrlClient.ObjectKey{Namespace: centralNamespace, Name: "central-htpasswd"}, &secret)
			return err == nil
		}, 10*time.Second, 5*time.Minute)
	if !exists {
		return "", errors.Wrapf(err, "waiting for admin password secret %s/central-htpasswd", centralNamespace)
	}

	// Retrieve the secret and the "password" key, additionally make sure to trim all spaces from the value.
	secret := corev1.Secret{} // pragma: allowlist secret
	if err := k8sClient.Get(ctx, ctrlClient.ObjectKey{Namespace: centralNamespace, Name: "central-htpasswd"}, &secret); err != nil {
		return "", errors.Wrapf(err, "retrieving secret %s/central-htpasswd", centralNamespace)
	}
	password := strings.TrimSpace(string(secret.Data["password"]))
	if password == "" {
		return "", errors.Errorf("admin password was empty for central instance %s/%s",
			centralNamespace, centralName)
	}

	// Wait for the first successful response from the central API using the basic auth provider.
	centralClient := centralClient.NewCentralClient(
		private.ManagedCentral{
			Metadata: private.ManagedCentralAllOfMetadata{Name: centralName, Namespace: centralNamespace},
		}, centralUIEndpoint, password)
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
		return "", errors.Errorf(
			"no successful request could be done with basic auth provider for central instance %s/%s",
			centralNamespace, centralName)
	}

	return password, nil
}

// DisableAdminPassword disables the admin password for the given central instance.
// This will be done by changing the central CR, and requires appropriate access to K8S.
func DisableAdminPassword(ctx context.Context, centralInstance string) error {
	client := k8s.CreateClientOrDie()

	centralNamespace := fmt.Sprintf("rhacs-%s", centralInstance)

	// Retrieve existing central.
	central := v1alpha1.Central{}
	err := client.Get(ctx,
		ctrlClient.ObjectKey{Namespace: centralNamespace, Name: centralInstance},
		&central,
	)
	if err != nil {
		return errors.Wrapf(err, "retrieving central instance %s/%s", centralNamespace, centralInstance)
	}

	// If admin password generation disabled is not set, default to true, since we need to explicitly set it in this
	// case to disable it.
	if pointer.BoolDeref(central.Spec.Central.AdminPasswordGenerationDisabled, true) {
		// Disable admin password generation, increase the revision, and update the central CR.
		central.Spec.Central.AdminPasswordGenerationDisabled = pointer.Bool(true)
		if err := increaseCentralRevision(&central); err != nil {
			return errors.Wrapf(err, "increasing central revision for central instance %s/%s",
				centralInstance, centralNamespace)
		}
		if err := client.Update(ctx, &central); err != nil {
			return errors.Wrapf(err, "updating central instance %s/%s", centralNamespace, centralInstance)
		}
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
