// Package operator provides install/upgrade logic for ACS Operator
package operator

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/charts"
	"helm.sh/helm/v3/pkg/chartutil"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	// ACSOperatorNamespace default Operator Namespace
	ACSOperatorNamespace     = "rhacs"
	releaseName              = "rhacs-operator"
	operatorDeploymentPrefix = "rhacs-operator"
)

// ACSOperatorManager keeps data necessary for managing ACS Operator
type ACSOperatorManager struct {
	client ctrlClient.Client
}

// InstallOrUpgrade provisions or upgrades an existing ACS Operator from helm chart template
func (u *ACSOperatorManager) InstallOrUpgrade(ctx context.Context, operators OperatorConfigs) error {
	objs, err := RenderChart(operators)
	if err != nil {
		return err
	}
	for _, obj := range objs {
		if obj.GetNamespace() == "" {
			obj.SetNamespace(ACSOperatorNamespace)
		}
		err := charts.InstallOrUpdateChart(ctx, obj, u.client)
		if err != nil {
			return fmt.Errorf("failed to update operator object %w", err)
		}
	}
	return nil
}

// RenderChart renders the operator helm chart manifests
func RenderChart(operators OperatorConfigs) ([]*unstructured.Unstructured, error) {
	if len(operators.Configs) == 0 {
		return nil, nil
	}
	var valuesArr []chartutil.Values
	for _, operator := range operators.Configs {
		valuesArr = append(valuesArr, chartutil.Values(operator))
	}

	resourcesChart, err := charts.GetChart("rhacs-operator", operators.CRDURLs)
	if err != nil {
		return nil, fmt.Errorf("failed getting chart: %w", err)
	}

	operatorValuesIntf, ok := resourcesChart.Values["operator"]
	if !ok {
		return nil, fmt.Errorf("failed getting operator values from chart")
	}
	operatorValues, ok := operatorValuesIntf.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("operator values are not a map")
	}
	defaultValuesIntf, ok := operatorValues["default"]
	if !ok {
		return nil, fmt.Errorf("failed getting default values from chart")
	}
	defaultValues, ok := defaultValuesIntf.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("default values are not a map")
	}

	chartVals := chartutil.Values{
		"operator": chartutil.Values{
			"images":  valuesArr,
			"default": chartutil.Values(defaultValues),
		},
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
func (u *ACSOperatorManager) RemoveUnusedOperators(ctx context.Context, operatorConfigs []OperatorConfig) error {
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
		found := false
		for _, expectedOperator := range operatorConfigs {
			if expectedOperator.GetDeploymentName() == deployment.GetName() {
				found = true
			}
		}
		if !found {
			unusedDeployments = append(unusedDeployments, deployment.GetName())
		}
	}

	if len(unusedDeployments) > 0 {
		glog.Infof("Detected %d unused deployments: %s.", len(unusedDeployments), strings.Join(unusedDeployments, ", "))
	}

	for _, deploymentName := range unusedDeployments {
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deploymentName,
				Namespace: ACSOperatorNamespace,
			},
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
