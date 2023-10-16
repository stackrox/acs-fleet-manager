// Package operator provides install/upgrade logic for ACS Operator
package operator

import (
	"bytes"
	"context"
	"fmt"

	"github.com/golang/glog"
	errors3 "github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/charts"
	"golang.org/x/exp/slices"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/releaseutil"
	"helm.sh/helm/v3/pkg/storage/driver"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/resource"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ACSOperatorNamespace default Operator Namespace
	ACSOperatorNamespace = "rhacs"
	// ACSOperatorConfigMap name for configMap with operator deployment configurations
	releaseName              = "rhacs-operator"
	operatorDeploymentPrefix = "rhacs-operator"
)

// ACSOperatorManager keeps data necessary for managing ACS Operator
type ACSOperatorManager struct {
	client ctrlClient.Client
}

// InstallOrUpgrade provisions or upgrades an existing ACS Operator from helm chart template
func (u *ACSOperatorManager) InstallOrUpgrade(ctx context.Context, operators OperatorConfigs) error {
	resourcesChart, err := charts.GetChart("rhacs-operator", operators.CRDURLs)
	if err != nil {
		return fmt.Errorf("failed getting chart: %w", err)
	}

	var operatorValues []chartutil.Values
	for _, operator := range operators.Configs {
		operatorValues = append(operatorValues, chartutil.Values(operator))
	}

	halmValues := chartutil.Values{
		"operator": chartutil.Values{
			"images": operatorValues,
		},
	}

	settings := cli.New()
	settings.SetNamespace(ACSOperatorNamespace)
	actionConfig := new(action.Configuration)

	if err := actionConfig.Init(settings.RESTClientGetter(), ACSOperatorNamespace, "secret", glog.Infof); err != nil {
		return fmt.Errorf("failed initializing action config: %w", err)
	}

	actionConfig.Releases.MaxHistory = 5

	history, err := actionConfig.Releases.History(releaseName)
	if err != nil {
		if !errors3.Is(err, driver.ErrReleaseNotFound) {
			return fmt.Errorf("failed listing existing releases: %w", err)
		}
	}

	var deployed *release.Release
	if len(history) > 0 {
		glog.Infof("Found %d previous releases", len(history))
		releaseutil.Reverse(history, releaseutil.SortByRevision)
		deployed = history[0]
	}

	metadataAccessor := meta.NewAccessor()

	if deployed != nil {
		glog.Infof("Previous release exists, checking if upgrade is needed")
		upgrade := action.NewUpgrade(actionConfig)
		upgrade.DryRun = true
		upgrade.Namespace = ACSOperatorNamespace
		wouldBeRelease, err := upgrade.Run("rhacs-operator", resourcesChart, halmValues)
		if err != nil {
			return fmt.Errorf("failed dry-run upgrading operator: %w", err)
		}
		if wouldBeRelease.Manifest == deployed.Manifest {
			glog.Infof("No changes detected for ACS Operators")
			return nil
		}
		glog.Infof("Changes detected for ACS Operators, upgrading")
		upgrade.DryRun = false
		newRelease, err := upgrade.Run("rhacs-operator", resourcesChart, halmValues)
		if err != nil {
			return fmt.Errorf("failed upgrading operator: %w", err)
		}

		target, err := actionConfig.KubeClient.Build(bytes.NewBufferString(newRelease.Manifest), false)
		if err != nil {
			return fmt.Errorf("failed building resources from manifest: %w", err)
		}
		original, err := actionConfig.KubeClient.Build(bytes.NewBufferString(deployed.Manifest), false)
		if err != nil {
			return fmt.Errorf("failed building resources from manifest: %w", err)
		}

		for _, info := range original.Difference(target) {
			glog.Infof("Deleting %s %q in namespace %s...", info.Mapping.GroupVersionKind.Kind, info.Name, info.Namespace)

			if err := info.Get(); err != nil {
				glog.Infof("Unable to get obj %q, err: %s", info.Name, err)
				continue
			}
			annotations, err := metadataAccessor.Annotations(info.Object)
			if err != nil {
				glog.Infof("Unable to get annotations on %q, err: %s", info.Name, err)
			}
			if annotations != nil && annotations[kube.ResourcePolicyAnno] == kube.KeepPolicy {
				glog.Infof("Skipping delete of %q due to annotation [%s=%s]", info.Name, kube.ResourcePolicyAnno, kube.KeepPolicy)
				continue
			}
			if err := deleteResource(info); err != nil {
				if !errors.IsNotFound(err) {
					glog.Infof("Failed to delete %q, err: %s", info.ObjectName(), err)
					continue
				}
			}
		}

	} else {
		glog.Infof("No previous release exists, installing")
		install := action.NewInstall(actionConfig)
		install.ReleaseName = releaseName
		install.Namespace = ACSOperatorNamespace
		install.DryRun = true

		newRelease, err := install.Run(resourcesChart, halmValues)
		if err != nil {
			return fmt.Errorf("failed dry-run installing operator: %w", err)
		}

		target, err := actionConfig.KubeClient.Build(bytes.NewBufferString(newRelease.Manifest), false)
		if err != nil {
			return fmt.Errorf("failed building resources from manifest: %w", err)
		}

		if visitErr := target.Visit(func(info *resource.Info, err error) error {

			if err != nil {
				return fmt.Errorf("failed visiting resources: %w", err)
			}

			// check if the resource already exists
			if err := info.Get(); err != nil {
				if !errors.IsNotFound(err) {
					return fmt.Errorf("failed getting resource: %w", err)
				}
				return nil
			}

			// delete the resource if it is not owned by Helm
			labels, err := metadataAccessor.Labels(info.Object)
			if err != nil {
				return fmt.Errorf("failed getting labels for resource: %w", err)
			}

			if labels["app.kubernetes.io/managed-by"] != "Helm" {
				if err := deleteResource(info); err != nil {
					if !errors.IsNotFound(err) {
						return fmt.Errorf("failed deleting resource: %w", err)
					}
				}
			}

			return nil

		}); visitErr != nil {
			return fmt.Errorf("failed visiting resources: %w", err)
		}

		_, err = install.Run(resourcesChart, halmValues)
		if err != nil {
			return fmt.Errorf("failed installing operator: %w", err)
		}
	}

	return nil
}

func deleteResource(info *resource.Info) error {
	policy := metav1.DeletePropagationBackground
	opts := &metav1.DeleteOptions{PropagationPolicy: &policy}
	_, err := resource.NewHelper(info.Client, info.Mapping).DeleteWithOptions(info.Namespace, info.Name, opts)
	return errors3.Wrap(err, "deleting resource")
}

// RenderChart renders the operator helm chart manifests
func RenderChart(operators OperatorConfigs) ([]*unstructured.Unstructured, error) {
	var valuesArr []chartutil.Values
	for _, operator := range operators.Configs {
		valuesArr = append(valuesArr, chartutil.Values(operator))
	}
	chartVals := chartutil.Values{
		"operator": chartutil.Values{
			"images": valuesArr,
		},
	}

	resourcesChart, err := charts.GetChart("rhacs-operator", operators.CRDURLs)
	if err != nil {
		return nil, fmt.Errorf("failed getting chart: %w", err)
	}

	releaseName := "rhacs-operator"
	settings := cli.New()
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), ACSOperatorNamespace, "secret", glog.Infof); err != nil {
		return nil, fmt.Errorf("failed initializing action config: %w", err)
	}
	if _, err := action.NewGet(actionConfig).Run(releaseName); err != nil {
		if !errors3.Is(err, driver.ErrReleaseNotFound) {
			return nil, fmt.Errorf("failed listing existing releases: %w", err)
		}
		_, err = action.NewInstall(actionConfig).Run(resourcesChart, chartVals)
		if err != nil {
			return nil, fmt.Errorf("failed installing operator: %w", err)
		}
	} else {
		if _, err := action.NewUpgrade(actionConfig).Run("rhacs-operator", resourcesChart, chartVals); err != nil {
			return nil, fmt.Errorf("failed upgrading operator: %w", err)
		}
	}

	objs, err := charts.RenderToObjects(releaseName, ACSOperatorNamespace, resourcesChart, chartVals)
	if err != nil {
		return nil, fmt.Errorf("failed rendering operator chart: %w", err)
	}
	return objs, nil
}

// ListVersionsWithReplicas returns currently running ACS Operator versions with number of ready replicas
func (u *ACSOperatorManager) ListVersionsWithReplicas(ctx context.Context) (map[string]int32, error) {
	deployments := &appsv1.DeploymentList{}
	labels := map[string]string{"app": "rhacs-operator"}
	err := u.client.List(ctx, deployments,
		ctrlClient.InNamespace(ACSOperatorNamespace),
		ctrlClient.MatchingLabels(labels),
	)
	if err != nil {
		return nil, fmt.Errorf("failed list operator deployments: %w", err)
	}

	versionWithReplicas := make(map[string]int32)
	for _, dep := range deployments.Items {
		for _, c := range dep.Spec.Template.Spec.Containers {
			if c.Name == "manager" {
				versionWithReplicas[c.Image] = dep.Status.ReadyReplicas
			}
		}
	}

	return versionWithReplicas, nil
}

// RemoveUnusedOperators removes unused operator deployments from the cluster. It receives a list of operator images which should be present in the cluster and removes all deployments which do not deploy any of the desired images.
func (u *ACSOperatorManager) RemoveUnusedOperators(ctx context.Context, desiredImages []string) error {
	deployments := &appsv1.DeploymentList{}
	labels := map[string]string{"app": "rhacs-operator"}
	err := u.client.List(ctx, deployments,
		ctrlClient.InNamespace(ACSOperatorNamespace),
		ctrlClient.MatchingLabels(labels),
	)
	if err != nil {
		return fmt.Errorf("failed list operator deployments: %w", err)
	}

	var unusedDeployments []string
	for _, deployment := range deployments.Items {
		for _, container := range deployment.Spec.Template.Spec.Containers {
			if container.Name == "manager" && !slices.Contains(desiredImages, container.Image) {
				unusedDeployments = append(unusedDeployments, deployment.Name)
			}
		}
	}

	for _, deploymentName := range unusedDeployments {
		deployment := &appsv1.Deployment{}
		err := u.client.Get(ctx, ctrlClient.ObjectKey{Namespace: ACSOperatorNamespace, Name: deploymentName}, deployment)
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("retrieving operator deployment %s: %w", deploymentName, err)
		}
		err = u.client.Delete(ctx, deployment)
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("deleting operator deployment %s: %w", deploymentName, err)
		}
	}

	return nil
}

// NewACSOperatorManager creates a new ACS Operator Manager
func NewACSOperatorManager(k8sClient ctrlClient.Client) *ACSOperatorManager {
	return &ACSOperatorManager{
		client: k8sClient,
	}
}
