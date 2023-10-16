package e2e

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/operator"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/gitops"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/pkg/features"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	namespace              = "acscs"
	gitopsConfigmapName    = "fleet-manager-gitops-config"
	gitopsConfigmapDataKey = "config.yaml"
)

var _ = Describe("Fleetshard-sync Targeted Upgrade", func() {
	var client *fleetmanager.Client

	ctx := context.Background()

	BeforeEach(func() {
		if !features.TargetedOperatorUpgrades.Enabled() {
			Skip("Skipping canary upgrade test")
		}
		option := fleetmanager.OptionFromEnv()
		auth, err := fleetmanager.NewStaticAuth(context.Background(), fleetmanager.StaticOption{StaticToken: option.Static.StaticToken})
		Expect(err).ToNot(HaveOccurred())
		client, err = fleetmanager.NewClient(fleetManagerEndpoint, auth)
		Expect(err).ToNot(HaveOccurred())
	})

	if fmEndpointEnv := os.Getenv("FLEET_MANAGER_ENDPOINT"); fmEndpointEnv != "" {
		fleetManagerEndpoint = fmEndpointEnv
	}

	operator1 := operator.OperatorConfig{
		"deploymentName":       "rhacs-operator-1",
		"image":                fmt.Sprintf("quay.io/rhacs-eng/stackrox-operator:%s", "4.2.0-395-gf9ab14f1bf-dirty"),
		"centralLabelSelector": fmt.Sprintf("rhacs.redhat.com/version-selector=%s", "operator-1"),
	}
	operator2 := operator.OperatorConfig{
		"deploymentName":       "rhacs-operator-2",
		"image":                fmt.Sprintf("quay.io/rhacs-eng/stackrox-operator:%s", "4.2.0-395-gf9ab14f1bf-dirty"),
		"centralLabelSelector": fmt.Sprintf("rhacs.redhat.com/version-selector=%s", "operator-2"),
	}

	Describe("Managing ACS operators", func() {

		BeforeEach(func() {
			Skip("")
		})

		It("can uninstall all operators", func() {
			var operatorConfigs []operator.OperatorConfig
			Expect(updateInstalledOperators(ctx, operatorConfigs)).To(Succeed())
			Eventually(checkDeployedOperators(ctx, operatorConfigs...), "5m", "1s", ctx).
				MustPassRepeatedly(10).
				Should(Succeed())
		})

		It("can install operator1", func() {
			operatorConfigs := []operator.OperatorConfig{operator1}
			Expect(updateInstalledOperators(ctx, operatorConfigs)).To(Succeed())
			Eventually(checkDeployedOperators(ctx, operatorConfigs...), "5m", "1s", ctx).
				MustPassRepeatedly(10).
				Should(Succeed())
		})

		It("can install both operators", func() {
			operatorConfigs := []operator.OperatorConfig{operator1, operator2}
			Expect(updateInstalledOperators(ctx, operatorConfigs)).To(Succeed())
			Eventually(checkDeployedOperators(ctx, operatorConfigs...), "5m", "1s", ctx).
				MustPassRepeatedly(10).
				Should(Succeed())
		})

		It("can install operator2", func() {
			operatorConfigs := []operator.OperatorConfig{operator2}
			Expect(updateInstalledOperators(ctx, operatorConfigs)).To(Succeed())
			Eventually(checkDeployedOperators(ctx, operatorConfigs...), "5m", "1s", ctx).
				MustPassRepeatedly(10).
				Should(Succeed())
		})

		It("can remove all operators again", func() {
			var operatorConfigs []operator.OperatorConfig
			Expect(updateInstalledOperators(ctx, operatorConfigs)).To(Succeed())
			Eventually(checkDeployedOperators(ctx, operatorConfigs...), "5m", "1s", ctx).
				MustPassRepeatedly(10).
				Should(Succeed())
		})
	})

	Describe("Managing Centrals", func() {

		var (
			central1 *public.CentralRequest
			central2 *public.CentralRequest
			err      error
		)

		It("creates 2 centrals", func() {
			central1, err = createCentral(ctx, client)
			Expect(err).ToNot(HaveOccurred())
			central2, err = createCentral(ctx, client)
			Expect(err).ToNot(HaveOccurred())
		})

		It("installs operator1", func() {
			operators := []operator.OperatorConfig{operator1}
			Expect(updateInstalledOperators(ctx, operators)).
				To(Succeed())
			Eventually(checkDeployedOperators(ctx, operators...), "5m", "1s", ctx).
				MustPassRepeatedly(10).
				Should(Succeed())
		})

		It("Sets the overrides so that operator1 reconciles all Centrals", func() {
			Expect(updateCentralOverrides(ctx,
				overrideAllCentralsToUseMinimalResources(),
				overrideAllCentralsToBeReconciledByOperator(operator1),
			)).
				To(Succeed())
		})

		It("checks that operator1 reconciles both Centrals", func() {
			Eventually(checkCentralsReconciledBy(ctx, central1, operator1, central2, operator1), "5m", "1s", ctx).
				MustPassRepeatedly(10).
				Should(Succeed())
		})

		It("installs a second operator", func() {
			operators := []operator.OperatorConfig{operator1, operator2}
			Expect(updateInstalledOperators(ctx, operators)).
				To(Succeed())
			Eventually(checkDeployedOperators(ctx, operators...), "5m", "1s", ctx).
				MustPassRepeatedly(10).
				Should(Succeed())
		})

		It("should not change the centrals", func() {
			Consistently(checkCentralsReconciledBy(ctx, central1, operator1, central2, operator1), "1m", "1s", ctx).
				Should(Succeed())
		})

		It("Sets the overrides so that operator2 reconciles central2", func() {
			Expect(updateCentralOverrides(ctx,
				overrideAllCentralsToUseMinimalResources(),
				overrideAllCentralsToBeReconciledByOperator(operator1),
				overrideOneCentralToBeReconciledByOperator(central2, operator2),
			)).To(Succeed())
		})

		It("checks that operator1 reconciles central1 and operator2 reconciles central2", func() {
			Eventually(checkCentralsReconciledBy(ctx, central1, operator1, central2, operator2), "5m", "1s", ctx).
				MustPassRepeatedly(20).
				Should(Succeed())
		})

		It("Sets the central overrides so that operator2 reconciles all centrals", func() {
			Expect(updateCentralOverrides(ctx,
				overrideAllCentralsToUseMinimalResources(),
				overrideAllCentralsToBeReconciledByOperator(operator2),
			)).
				To(Succeed())
		})

		It("checks that operator2 reconciles all centrals", func() {
			Eventually(checkCentralsReconciledBy(ctx, central1, operator2, central2, operator2), "5m", "1s", ctx).
				MustPassRepeatedly(10).
				Should(Succeed())
		})

		It("uninstalls the operators", func() {
			var operators []operator.OperatorConfig
			Expect(updateInstalledOperators(ctx, operators)).
				To(Succeed())
			Eventually(checkDeployedOperators(ctx, operators...), "5m", "1s", ctx).
				MustPassRepeatedly(10).
				Should(Succeed())
		})

		It("should not change the centrals", func() {
			Consistently(checkCentralsReconciledBy(ctx, central1, operator2, central2, operator2), "1m", "1s", ctx).
				Should(Succeed())
		})

		It("reinstalls the operators", func() {
			operators := []operator.OperatorConfig{operator1, operator2}
			Expect(updateInstalledOperators(ctx, operators)).
				To(Succeed())
			Eventually(checkDeployedOperators(ctx, operators...), "5m", "1s", ctx).
				MustPassRepeatedly(10).
				Should(Succeed())
		})

		It("should not change the centrals", func() {
			Consistently(checkCentralsReconciledBy(ctx, central1, operator2, central2, operator2), "1m", "1s", ctx).
				Should(Succeed())
		})

		Describe("When GitOps configuration becomes invalid", func() {

			var backup string

			It("sets an invalid gitops config", func() {
				backup, err = getGitopsYaml(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(updateGitopsConfigRaw(ctx, "invalid yaml")).To(Succeed())
			})

			It("should not change the centrals nor operators", func() {
				Consistently(
					allOf(
						checkCentralsReconciledBy(ctx, central1, operator2, central2, operator2),
						checkDeployedOperators(ctx, operator1, operator2),
					),
					"1m", "1s", ctx).
					Should(Succeed())
			})

			It("restarts fleet-manager", func() {
				pods, err := k8sClientSet.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
					LabelSelector: "application=fleet-manager",
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(len(pods.Items)).To(Equal(1))
				pod := pods.Items[0]
				Expect(k8sClientSet.CoreV1().Pods(namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})).To(Succeed())
				Eventually(func() bool {
					_, err := k8sClientSet.CoreV1().Pods(namespace).Get(ctx, pod.Name, metav1.GetOptions{})
					return apiErrors.IsNotFound(err)
				}, "1m", "1s", ctx).Should(BeTrue())
				Eventually(func() bool {
					pod, err := k8sClientSet.CoreV1().Pods(namespace).Get(ctx, pod.Name, metav1.GetOptions{})
					if err != nil {
						return false
					}
					return pod.Status.Phase == v1.PodRunning
				}, "1m", "1s", ctx)
			})

			It("should not change the centrals nor operators", func() {
				Consistently(
					allOf(
						checkCentralsReconciledBy(ctx, central1, operator2, central2, operator2),
						checkDeployedOperators(ctx, operator1, operator2),
					),
					"1m", "1s", ctx).
					Should(Succeed())
			})

			It("restores the valid gitops config", func() {
				Expect(updateGitopsConfigRaw(ctx, backup)).
					To(Succeed())
			})

			It("should not change the centrals nor operators", func() {
				Consistently(
					allOf(
						checkCentralsReconciledBy(ctx, central1, operator2, central2, operator2),
						checkDeployedOperators(ctx, operator1, operator2),
					),
					"1m", "1s", ctx).
					Should(Succeed())
			})

		})
	})
})

func updateInstalledOperators(ctx context.Context, operatorConfigs []operator.OperatorConfig) error {
	return updateGitopsConfig(ctx, func(cfg gitops.Config) gitops.Config {
		if operatorConfigs == nil {
			operatorConfigs = make([]operator.OperatorConfig, 0)
		}
		cfg.RHACSOperators.Configs = operatorConfigs
		return cfg
	})
}

func overrideAllCentralsToBeReconciledByOperator(operatorConfig operator.OperatorConfig) gitops.CentralOverride {
	return overrideAllCentralsWithPatch(reconciledByOperatorPatch(operatorConfig))
}

func overrideAllCentralsToUseMinimalResources() gitops.CentralOverride {
	return overrideAllCentralsWithPatch(minimalCentralResourcesPatch())
}

func overrideOneCentralToBeReconciledByOperator(central *public.CentralRequest, operatorConfig operator.OperatorConfig) gitops.CentralOverride {
	return gitops.CentralOverride{
		InstanceIDs: []string{central.Id},
		Patch:       reconciledByOperatorPatch(operatorConfig),
	}
}

func overrideAllCentralsWithPatch(patch string) gitops.CentralOverride {
	return gitops.CentralOverride{
		InstanceIDs: []string{"*"},
		Patch:       patch,
	}
}

func reconciledByOperatorPatch(operatorConfig operator.OperatorConfig) string {
	key, value, err := getLabelAndVersionFromOperatorConfig(operatorConfig)
	if err != nil {
		panic(err)
	}
	return centralLabelPatch(key, value)
}

func minimalCentralResourcesPatch() string {
	return `
spec:
  monitoring:
    openshift:
      enabled: false
  central:
    db:
      resources:
        limits:
          cpu: 500m
          memory: 500Mi
        requests:
          cpu: 100m
          memory: 100Mi
    resources:
      limits:
        cpu: 500m
        memory: 500Mi
      requests:
        cpu: 100m
        memory: 100Mi
  scanner:
    analyzer:
      resources:
        limits:
          cpu: 1000m
          memory: 1000Mi
        requests:
          cpu: 100m
          memory: 100Mi
      scaling:
        autoScaling: "Disabled"
        replicas: 1
    db:
      resources:
        limits:
          cpu: 1000m
          memory: 1000Mi
        requests:
          cpu: 100m
          memory: 100Mi
`
}

func centralLabelPatch(key, value string) string {
	return fmt.Sprintf(`
metadata:
  labels:
    ` + key + `: "` + value + `"`)
}

func updateCentralOverrides(ctx context.Context, overrides ...gitops.CentralOverride) error {
	return updateGitopsConfig(ctx, func(cfg gitops.Config) gitops.Config {
		cfg.Centrals.Overrides = overrides
		return cfg
	})
}

func updateGitopsConfig(ctx context.Context, updateFn func(cfg gitops.Config) gitops.Config) error {

	var configMap v1.ConfigMap
	if err := k8sClient.Get(ctx, ctrlClient.ObjectKey{Namespace: namespace, Name: gitopsConfigmapName}, &configMap); err != nil {
		return err
	}

	var cfg gitops.Config
	if err := yaml.Unmarshal([]byte(configMap.Data[gitopsConfigmapDataKey]), &cfg); err != nil {
		return err
	}

	cfg = updateFn(cfg)

	newYaml, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	configMap.Data[gitopsConfigmapDataKey] = string(newYaml)

	err = k8sClient.Update(ctx, &configMap)
	if err != nil && !apiErrors.IsNotFound(err) {
		return err
	}

	return nil
}

func getGitopsConfigMap(ctx context.Context) (*v1.ConfigMap, error) {
	var configMap v1.ConfigMap
	if err := k8sClient.Get(ctx, ctrlClient.ObjectKey{Namespace: namespace, Name: gitopsConfigmapName}, &configMap); err != nil {
		return nil, err
	}
	return &configMap, nil
}

func getGitopsYaml(ctx context.Context) (string, error) {
	configMap, err := getGitopsConfigMap(ctx)
	if err != nil {
		return "", err
	}
	return configMap.Data[gitopsConfigmapDataKey], nil
}

func updateGitopsConfigRaw(ctx context.Context, newValue string) error {
	var configMap v1.ConfigMap
	if err := k8sClient.Get(ctx, ctrlClient.ObjectKey{Namespace: namespace, Name: gitopsConfigmapName}, &configMap); err != nil {
		return err
	}
	configMap.Data[gitopsConfigmapDataKey] = newValue
	return k8sClient.Update(ctx, &configMap)
}

func getOperatorDeployments(ctx context.Context) ([]appsv1.Deployment, error) {
	deployments := &appsv1.DeploymentList{}
	labels := map[string]string{"app": "rhacs-operator"}
	err := k8sClient.List(ctx, deployments,
		ctrlClient.InNamespace(operator.ACSOperatorNamespace),
		ctrlClient.MatchingLabels(labels))
	if err != nil {
		return nil, err
	}
	return deployments.Items, nil
}

func getCentralCR(ctx context.Context, name string, namespace string) (*v1alpha1.Central, error) {
	central := &v1alpha1.Central{}
	centralKey := ctrlClient.ObjectKey{Namespace: namespace, Name: name}
	err := k8sClient.Get(ctx, centralKey, central)
	if err != nil {
		return nil, err
	}
	return central, nil
}

func getDeployment(ctx context.Context, namespace string, name string) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	err := k8sClient.Get(ctx, ctrlClient.ObjectKey{Namespace: namespace, Name: name}, deployment)
	return deployment, err
}

func getContainer(name string, containers []v1.Container) (*v1.Container, error) {
	for _, container := range containers {
		if container.Name == name {
			return &container, nil
		}
	}
	return nil, fmt.Errorf("expected container %s to exist", name)
}

// checkDeployedOperators validates that the given operator configurations are deployed
func checkDeployedOperators(ctx context.Context, desiredConfigs ...operator.OperatorConfig) func() error {
	return func() error {
		operatorDeployments, err := getOperatorDeployments(ctx)
		if err != nil {
			return err
		}

		deploymentsByName := map[string]appsv1.Deployment{}
		for _, operatorDeployment := range operatorDeployments {
			deploymentsByName[operatorDeployment.Name] = operatorDeployment
		}

		if len(operatorDeployments) != len(desiredConfigs) {
			var desiredOperatorNames []string
			var foundOperatorNames []string
			var missingOperatorNames []string
			var extraOperatorNameMap = map[string]struct{}{}
			for _, operatorDeployment := range operatorDeployments {
				foundOperatorNames = append(foundOperatorNames, operatorDeployment.Name)
				extraOperatorNameMap[operatorDeployment.Name] = struct{}{}
			}
			for _, desiredConfig := range desiredConfigs {
				desiredOperatorName := desiredConfig.GetDeploymentName()
				desiredOperatorNames = append(desiredOperatorNames, desiredOperatorName)
				if _, ok := extraOperatorNameMap[desiredOperatorName]; !ok {
					missingOperatorNames = append(missingOperatorNames, desiredOperatorName)
				} else {
					delete(extraOperatorNameMap, desiredOperatorName)
				}
			}
			var extraOperatorNames []string
			for extraOperatorName := range extraOperatorNameMap {
				extraOperatorNames = append(extraOperatorNames, extraOperatorName)
			}
			return fmt.Errorf("expected %v operator deployments, got %v: extra operators %v, missing operators: %v", desiredOperatorNames, foundOperatorNames, extraOperatorNames, missingOperatorNames)
		}

		for _, desiredConfig := range desiredConfigs {
			// check that it is the correct operator
			desiredDeploymentName := desiredConfig.GetDeploymentName()

			deployment, ok := deploymentsByName[desiredDeploymentName]
			if !ok {
				return fmt.Errorf("operator deployment %s not found", desiredDeploymentName)
			}

			var assertions []deploymentAssertion

			// check that the image is the correct one
			assertions = append(assertions, operatorHasImage(desiredConfig.GetImage()))

			// check that the deployment is ready
			assertions = append(assertions, deploymentIsReady())

			// check the central label selector
			assertions = append(assertions, operatorHasCentralLabelSelector(desiredConfig.GetCentralLabelSelector()))

			// check the secured cluster label selector
			assertions = append(assertions, operatorHasSecuredClusterLabelSelector(desiredConfig.GetSecuredClusterLabelSelector()))

			// check the central reconciler enabled
			assertions = append(assertions, operatorHasCentralReconcilerEnabled(desiredConfig.GetCentralReconcilerEnabled()))

			// check the secured cluster reconciler enabled
			assertions = append(assertions, operatorHasSecuredClusterReconcilerEnabled(desiredConfig.GetSecuredClusterReconcilerEnabled()))

			// validate the deployment
			if err := validateOperatorDeployment(&deployment, assertions...); err != nil {
				fmt.Println(err.Error())
				return err
			}
		}

		return nil
	}
}

// ASSERTION UTILS

func createCentral(ctx context.Context, client *fleetmanager.Client) (*public.CentralRequest, error) {
	createdCentral, _, err := client.PublicAPI().CreateCentral(ctx, true, public.CentralRequestPayload{
		CloudProvider: dpCloudProvider,
		MultiAz:       true,
		Name:          newCentralName(),
		Region:        dpRegion,
	})
	if err != nil {
		return nil, err
	}
	if createdCentral.Status != constants.CentralRequestStatusAccepted.String() {
		return nil, fmt.Errorf("expected central request to be accepted, got %s", createdCentral.Status)
	}
	return &createdCentral, err
}

func getLabelAndVersionFromOperatorConfig(operatorConfig operator.OperatorConfig) (string, string, error) {
	selector := operatorConfig.GetCentralLabelSelector()
	selectorParts := strings.Split(selector, "=")
	if len(selectorParts) != 2 {
		return "", "", fmt.Errorf("invalid selector %s", selector)
	}
	versionLabelKey := selectorParts[0]
	versionLabelValue := selectorParts[1]
	return versionLabelKey, versionLabelValue, nil
}

func checkCentralsReconciledBy(ctx context.Context, centralAndConfigs ...interface{}) func() error {
	return func() error {
		if len(centralAndConfigs)%2 != 0 {
			return fmt.Errorf("expected an even number of arguments, got %d", len(centralAndConfigs))
		}
		var centrals []*public.CentralRequest
		var operators []operator.OperatorConfig
		for i := 0; i < len(centralAndConfigs); i += 2 {
			central, ok := centralAndConfigs[i].(*public.CentralRequest)
			if !ok {
				return fmt.Errorf("expected argument %d to be a central, got %v", i, reflect.TypeOf(centralAndConfigs[i]))
			}
			centrals = append(centrals, central)
			operatorConfig, ok := centralAndConfigs[i+1].(operator.OperatorConfig)
			if !ok {
				return fmt.Errorf("expected argument %d to be an operator config, got %v", i+1, reflect.TypeOf(centralAndConfigs[i+1]))
			}
			operators = append(operators, operatorConfig)
		}
		for i := 0; i < len(centrals); i++ {
			//goland:noinspection GoNilness
			if err := checkCentralMatches(ctx, centrals[i], operators[i]); err != nil {
				return err
			}
		}
		return nil
	}
}

func checkCentralMatches(ctx context.Context, centralRequest *public.CentralRequest, operatorConfig operator.OperatorConfig) error {
	versionLabelKey, versionLabelValue, err := getLabelAndVersionFromOperatorConfig(operatorConfig)
	if err != nil {
		return err
	}

	imageParts := strings.Split(operatorConfig.GetImage(), ":")
	if len(imageParts) != 2 {
		panic(fmt.Errorf("invalid image %s", operatorConfig.GetImage()))
	}

	centralNamespace, err := services.FormatNamespace(centralRequest.Id)
	if err != nil {
		return err
	}

	central, err := getCentralCR(ctx, centralRequest.Name, centralNamespace)
	if err != nil {
		return errors.Wrap(err, "failed getting central CR")
	}

	if central.Labels == nil {
		return fmt.Errorf("expected central to have labels, got none")
	}
	actualVersionLabel, ok := central.Labels[versionLabelKey]
	if !ok {
		return fmt.Errorf("expected central to have label %s=%s, got no label", versionLabelKey, versionLabelValue)
	}
	if actualVersionLabel != versionLabelValue {
		return fmt.Errorf("expected central to have label %s=%s, got %s=%s", versionLabelKey, versionLabelValue, versionLabelKey, actualVersionLabel)
	}

	return nil
}

func checkCentralCRFor(ctx context.Context, centralRequest *public.CentralRequest, assertions ...centralAssertion) error {
	namespace, err := services.FormatNamespace(centralRequest.Id)
	if err != nil {
		return err
	}
	return checkCentralCR(ctx, centralRequest.Name, namespace, assertions...)
}

func checkCentralCR(ctx context.Context, name, namespace string, assertions ...centralAssertion) error {
	central, err := getCentralCR(ctx, name, namespace)
	if err != nil {
		return errors.Wrap(err, "failed getting central CR")
	}
	for _, assertion := range assertions {
		if err := assertion(central); err != nil {
			return err
		}
	}
	return nil
}

type centralAssertion func(central *v1alpha1.Central) error

var centralHasLabel = func(key, value string) centralAssertion {
	return func(central *v1alpha1.Central) error {
		existing, ok := central.Labels[key]
		if !ok {
			return fmt.Errorf("expected central to have label %s=%s, got no label", key, value)
		}
		if existing != value {
			return fmt.Errorf("expected central to have label %s=%s, got %s=%s", key, value, key, central.Labels[key])
		}
		return nil
	}
}

// deploymentAssertion is a function that validates an operator deployment
type deploymentAssertion func(deploy *appsv1.Deployment) error

// assertionAnyOf validates that at least one of the given assertions passes
func assertionAnyOf(assertions ...deploymentAssertion) deploymentAssertion {
	return func(deploy *appsv1.Deployment) error {
		var errs []error
		for _, assertion := range assertions {
			err := assertion(deploy)
			if err != nil {
				errs = append(errs, err)
			} else {
				return nil
			}
		}
		return fmt.Errorf("none of the assertions passed, got errors: %v", errs)
	}
}

var containerHasImage = func(container, image string) deploymentAssertion {
	return func(deploy *appsv1.Deployment) error {
		container, err := getContainer(container, deploy.Spec.Template.Spec.Containers)
		if err != nil {
			return err
		}
		if container.Image != image {
			return fmt.Errorf("expected container %s to have image %s, got %s", container.Name, image, container.Image)
		}
		return nil
	}
}

// operatorHasImage validates that the operator deployment has the given image
var operatorHasImage = func(image string) deploymentAssertion {
	return containerHasImage("manager", image)
}

var containerDoesntHaveEnvVar = func(container, name string) deploymentAssertion {
	return func(deploy *appsv1.Deployment) error {
		container, err := getContainer(container, deploy.Spec.Template.Spec.Containers)
		if err != nil {
			return err
		}
		for _, envVar := range container.Env {
			if envVar.Name == name {
				return fmt.Errorf("expected env var %s to not exist, got %v", name, envVar)
			}
		}
		return nil
	}
}

// operatorDoesntHaveEnvVar validates that the operator deployment doesn't have the given env var
var operatorDoesntHaveEnvVar = func(name string) deploymentAssertion {
	return containerDoesntHaveEnvVar("manager", name)
}

var containerHasEnvVar = func(container string, envVar v1.EnvVar) deploymentAssertion {
	return func(deploy *appsv1.Deployment) error {
		container, err := getContainer(container, deploy.Spec.Template.Spec.Containers)
		if err != nil {
			return err
		}
		for _, containerEnvVar := range container.Env {
			if containerEnvVar.Name == envVar.Name {
				if !reflect.DeepEqual(containerEnvVar, envVar) {
					return fmt.Errorf("expected env var %s to be %v, got %v", envVar.Name, envVar, containerEnvVar)
				}
			}
		}
		return nil
	}
}

var deploymentHasLabel = func(key, value string) deploymentAssertion {
	return func(deploy *appsv1.Deployment) error {
		if deploy.Labels == nil {
			return fmt.Errorf("expected deployment to have label %s=%s, got no labels", key, value)
		}
		existing, ok := deploy.Labels[key]
		if !ok {
			return fmt.Errorf("expected deployment to have label %s=%s, got no label", key, value)
		}
		if existing != value {
			return fmt.Errorf("expected deployment to have label %s=%s, got %s=%s", key, value, key, deploy.Labels[key])
		}
		return nil
	}
}

// operatorHasEnvVar validates that the operator deployment has the given env var
var operatorHasEnvVar = func(envVar v1.EnvVar) deploymentAssertion {
	return containerHasEnvVar("manager", envVar)
}

// operatorHasCentralLabelSelector validates that the operator deployment has the given central label selector
var operatorHasCentralLabelSelector = func(labelSelector string) deploymentAssertion {
	if len(labelSelector) == 0 {
		return assertionAnyOf(
			operatorHasEnvVar(v1.EnvVar{Name: "CENTRAL_LABEL_SELECTOR", Value: ""}),
			operatorDoesntHaveEnvVar("CENTRAL_LABEL_SELECTOR"),
		)
	}
	return operatorHasEnvVar(v1.EnvVar{Name: "CENTRAL_LABEL_SELECTOR", Value: labelSelector})
}

// operatorHasSecuredClusterLabelSelector validates that the operator deployment has the given secured cluster label selector
var operatorHasSecuredClusterLabelSelector = func(labelSelector string) deploymentAssertion {
	if len(labelSelector) == 0 {
		return assertionAnyOf(
			operatorHasEnvVar(v1.EnvVar{Name: "SECURED_CLUSTER_LABEL_SELECTOR", Value: ""}),
			operatorDoesntHaveEnvVar("SECURED_CLUSTER_LABEL_SELECTOR"),
		)
	}
	return operatorHasEnvVar(v1.EnvVar{Name: "SECURED_CLUSTER_LABEL_SELECTOR", Value: labelSelector})
}

// operatorHasSecuredClusterReconcilerEnabled validates that the operator deployment has the given secured cluster reconciler enabled
var operatorHasSecuredClusterReconcilerEnabled = func(enabled bool) deploymentAssertion {
	if enabled {
		return assertionAnyOf(
			operatorHasEnvVar(v1.EnvVar{Name: "SECURED_CLUSTER_RECONCILER_ENABLED", Value: "true"}),
			operatorDoesntHaveEnvVar("SECURED_CLUSTER_RECONCILER_ENABLED"),
		)
	}
	return operatorHasEnvVar(v1.EnvVar{Name: "SECURED_CLUSTER_RECONCILER_ENABLED", Value: "false"})
}

// operatorHasCentralReconcilerEnabled validates that the operator deployment has the given central reconciler enabled
var operatorHasCentralReconcilerEnabled = func(enabled bool) deploymentAssertion {
	if enabled {
		return assertionAnyOf(
			operatorHasEnvVar(v1.EnvVar{Name: "CENTRAL_RECONCILER_ENABLED", Value: "true"}),
			operatorDoesntHaveEnvVar("CENTRAL_RECONCILER_ENABLED"),
		)
	}
	return operatorHasEnvVar(v1.EnvVar{Name: "CENTRAL_RECONCILER_ENABLED", Value: "false"})
}

// deploymentIsReady validates that the operator deployment is ready
var deploymentIsReady = func() deploymentAssertion {
	return func(deploy *appsv1.Deployment) error {
		if deploy.Status.ReadyReplicas != 1 {
			return fmt.Errorf("operator deployment %s is not ready", deploy.Name)
		}
		if deploy.ObjectMeta.DeletionTimestamp != nil {
			return fmt.Errorf("operator deployment %s is being deleted", deploy.Name)
		}
		return nil
	}
}

// validateOperatorDeployment validates that the given operator deployment passes all the given assertions
func validateOperatorDeployment(deployment *appsv1.Deployment, assertions ...deploymentAssertion) error {
	for _, assertion := range assertions {
		err := assertion(deployment)
		if err != nil {
			return err
		}
	}
	return nil
}

func allOf(fns ...func() error) error {
	for _, fn := range fns {
		if err := fn(); err != nil {
			return err
		}
	}
	return nil
}

func restartDeployment(ctx context.Context, namespace, name string) error {
	pods, err := k8sClientSet.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", name),
	})
	if err != nil {
		return err
	}
	gotNames := map[string]struct{}{}
	for _, pod := range pods.Items {
		gotNames[pod.Name] = struct{}{}
		if err := k8sClientSet.CoreV1().Pods(namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{}); err != nil {
			if !apiErrors.IsNotFound(err) {
				return err
			}
		}
	}
	// check that all pods are gone
	wg := sync.WaitGroup{}
	for _, pod := range pods.Items {
		wg.Add(1)
		go func(podName string) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					_, err := k8sClientSet.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
					if apiErrors.IsNotFound(err) {
						return
					}
				}
			}
		}(pod.Name)
	}

	// wait for all pods to be gone
	wg.Wait()

	return nil
}
