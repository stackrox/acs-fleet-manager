package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	argocd "github.com/stackrox/acs-fleet-manager/pkg/argocd/apis/application/v1alpha1"
	"github.com/stackrox/rox/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	verticalpodautoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/stackrox/acs-fleet-manager/e2e/testutil"
	"github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/gitops"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	fmImpl "github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager/impl"
	"github.com/stackrox/acs-fleet-manager/pkg/features"
)

const (
	namespace              = "rhacs"
	gitopsConfigmapName    = "gitops-config"
	gitopsConfigmapDataKey = "config.yaml"
	operatorVersion1       = "4.5.4"
	operatorVersion2       = "4.5.3"
)

var (
	operatorConfig1         = operatorConfigForVersion(operatorVersion1)
	operatorConfig2         = operatorConfigForVersion(operatorVersion2)
	operator1DeploymentName = getDeploymentName(operatorVersion1)
	operator2DeploymentName = getDeploymentName(operatorVersion2)
)

var _ = Describe("Fleetshard-sync Targeted Upgrade", Ordered, func() {
	var (
		client *fleetmanager.Client
		ctx    = context.Background()
	)

	BeforeAll(func() {
		Expect(restoreDefaultGitopsConfig()).To(Succeed())
	})

	AfterAll(func() {
		Expect(restoreDefaultGitopsConfig()).To(Succeed())
	})

	BeforeEach(func() {
		testutil.SkipIf(!features.TargetedOperatorUpgrades.Enabled() || !runCanaryUpgradeTests, "Skipping canary upgrade test")
		option := fmImpl.OptionFromEnv()
		auth, err := fmImpl.NewStaticAuth(ctx, fmImpl.StaticOption{StaticToken: option.Static.StaticToken})
		Expect(err).ToNot(HaveOccurred())
		client, err = fmImpl.NewClient(fleetManagerEndpoint, auth)
		Expect(err).ToNot(HaveOccurred())
	})

	if fmEndpointEnv := os.Getenv("FLEET_MANAGER_ENDPOINT"); fmEndpointEnv != "" {
		fleetManagerEndpoint = fmEndpointEnv
	}

	Describe("should run ACS operators", Ordered, func() {

		It("should deploy operator 1 "+operator1DeploymentName, func() {
			// update gitops config to install one operator
			config := defaultGitopsConfig()
			config.Applications = []argocd.Application{
				crdsApplication(),
				operatorsApplication(operatorConfig1),
			}

			Expect(putGitopsConfig(ctx, config)).To(Succeed())
			Eventually(assertDeployedOperators(ctx, operator1DeploymentName)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should run operator 1 with central label selector "+operatorConfig1.CentralLabelSelector, func() {
			Eventually(operatorMatchesConfig(ctx, operatorConfig1)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should deploy two operators in different versions", func() {
			// add a second operator version to the gitops config
			config := defaultGitopsConfig()
			config.Applications = []argocd.Application{
				crdsApplication(),
				operatorsApplication(operatorConfig1, operatorConfig2),
			}

			Expect(putGitopsConfig(ctx, config)).To(Succeed())
			Eventually(assertDeployedOperators(ctx, operator1DeploymentName, operator2DeploymentName)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should deploy operator 1 with label selector "+operatorConfig1.CentralLabelSelector, func() {
			Eventually(operatorMatchesConfig(ctx, operatorConfig1)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should deploy operator 2 with label selector "+operatorConfig2.CentralLabelSelector, func() {
			Eventually(operatorMatchesConfig(ctx, operatorConfig2)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should delete operator 2 and only run operator 1", func() {
			config := defaultGitopsConfig()
			config.Applications = []argocd.Application{
				crdsApplication(),
				operatorsApplication(operatorConfig1),
			}

			Expect(putGitopsConfig(ctx, config)).To(Succeed())
			Eventually(assertDeployedOperators(ctx, operator1DeploymentName)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should deploy operator 1 with label selector "+operatorConfig1.CentralLabelSelector, func() {
			Eventually(operatorMatchesConfig(ctx, operatorConfig1)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})
	})

	Describe("should upgrade the central", Ordered, func() {
		ctx := ctx
		var createdCentral *public.CentralRequest
		var centralNamespace string

		It("run only one operator with version: "+operatorVersion1, func() {
			Expect(updateGitopsConfig(ctx, func(config gitops.Config) gitops.Config {
				config = defaultGitopsConfig()
				config.Applications = []argocd.Application{
					crdsApplication(),
					operatorsApplication(operatorConfig1),
				}
				config.TenantResources.Overrides = []gitops.TenantResourceOverride{
					overrideAllCentralsToBeReconciledByOperator(operatorConfig1),
					overrideAllCentralsToUseMinimalResources(),
				}
				return config
			})).To(Succeed())
			debugGitopsConfig(ctx)
			Eventually(assertDeployedOperators(ctx, operator1DeploymentName)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("creates central", func() {
			centralName := newCentralName()
			request := public.CentralRequestPayload{
				CloudProvider: dpCloudProvider,
				MultiAz:       true,
				Name:          centralName,
				Region:        dpRegion,
			}
			resp, _, err := client.PublicAPI().CreateCentral(ctx, true, request)
			Expect(err).To(Not(HaveOccurred()))
			createdCentral = &resp
			Expect(err).To(Not(HaveOccurred()))
			Expect(constants.CentralRequestStatusAccepted.String()).To(Equal(createdCentral.Status))
			centralNamespace, err = services.FormatNamespace(createdCentral.Id)
			debugGitopsConfig(ctx)
			Eventually(assertCentralLabelSelectorPresent(ctx, centralNamespace, operatorVersion1)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("upgrade central", func() {
			Expect(updateGitopsConfig(ctx, func(config gitops.Config) gitops.Config {
				config = defaultGitopsConfig()
				config.Applications = []argocd.Application{
					crdsApplication(),
					operatorsApplication(operatorConfig1, operatorConfig2),
				}
				config.TenantResources.Overrides = []gitops.TenantResourceOverride{
					overrideAllCentralsToBeReconciledByOperator(operatorConfig2),
					overrideAllCentralsToUseMinimalResources(),
				}
				return config
			})).To(Succeed())
			debugGitopsConfig(ctx)
			Eventually(assertCentralLabelSelectorPresent(ctx, centralNamespace, operatorVersion2)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("deploys an autoscaler", func() {
			_, err := getVPA(ctx, centralNamespace, "central-vpa")
			Expect(err).To(HaveOccurred())
			Expect(k8sErrors.IsNotFound(err)).To(BeTrue(), "central-vpa VerticalPodAutoscaler should not exist: %v", err)
			Expect(updateGitopsConfig(ctx, func(config gitops.Config) gitops.Config {
				config = defaultGitopsConfig()
				config.TenantResources.Overrides = append(config.TenantResources.Overrides, overrideCentralWithPatch(createdCentral.Id, tenantResourcesWithCentralVpaEnabled()))
				return config
			})).To(Succeed())
			debugGitopsConfig(ctx)
			Eventually(func() error {
				_, err := getVPA(ctx, centralNamespace, "central-vpa")
				return err
			}).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("removes the autoscaler", func() {
			_, err := getVPA(ctx, centralNamespace, "central-vpa")
			Expect(err).ToNot(HaveOccurred(), "central-vpa VerticalPodAutoscaler should exist: %v", err)
			Expect(updateGitopsConfig(ctx, func(config gitops.Config) gitops.Config {
				config = defaultGitopsConfig()
				return config
			})).To(Succeed())
			debugGitopsConfig(ctx)
			Eventually(func() error {
				_, err := getVPA(ctx, centralNamespace, "central-vpa")
				if !k8sErrors.IsNotFound(err) {
					return fmt.Errorf("vpa not removed")
				}
				return nil
			}).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("delete central", func() {
			Expect(deleteCentralByID(ctx, client, createdCentral.Id)).
				To(Succeed())
			Eventually(testutil.AssertCentralRequestDeprovisioning(ctx, client, createdCentral.Id)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})
	})
})

func assertCentralLabelSelectorPresent(ctx context.Context, centralNamespace, version string) func() error {
	return func() error {
		var centralCR v1alpha1.Central
		if err := assertCentralCRExists(ctx, &centralCR, centralNamespace)(); err != nil {
			return fmt.Errorf("failed finding central CR in namespace %s: %w", centralNamespace, err)
		}
		if centralCR.Labels == nil {
			return fmt.Errorf("central CR in namespace %s has no labels", centralNamespace)
		}
		if centralCR.GetLabels()["rhacs.redhat.com/version-selector"] != version {
			return fmt.Errorf("central CR in namespace %s has incorrect label selector %s", centralNamespace, centralCR.GetLabels()["rhacs.redhat.com/version-selector"])
		}
		return nil
	}
}

func assertDeployedOperators(ctx context.Context, expectedDeploymentNames ...string) func() error {
	return func() error {
		deployments, err := getOperatorDeployments(ctx)
		if err != nil {
			return err
		}
		wantSet := sets.NewString(expectedDeploymentNames...)
		actualSet := sets.NewString()
		for _, deployment := range deployments {
			actualSet.Insert(deployment.GetName())
		}
		extraSet := actualSet.Difference(wantSet)
		missingSet := wantSet.Difference(actualSet)
		if !actualSet.Equal(wantSet) {
			return fmt.Errorf("expected deployments %v. actual deployments %v. extra deployments: %v. missing deployments: %v",
				expectedDeploymentNames,
				actualSet.List(),
				extraSet.List(),
				missingSet.List(),
			)
		}
		return nil
	}
}

func putGitopsConfig(ctx context.Context, config gitops.Config) error {
	configYAML, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      gitopsConfigmapName,
		},
		Data: map[string]string{
			gitopsConfigmapDataKey: string(configYAML),
		},
	}
	if err := k8sClient.Update(ctx, configMap); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return err
		}
	} else {
		return nil
	}
	return k8sClient.Create(ctx, configMap)
}

func debugGitopsConfig(ctx context.Context) {
	var configMap v1.ConfigMap
	if err := k8sClient.Get(ctx, ctrlClient.ObjectKey{Namespace: namespace, Name: gitopsConfigmapName}, &configMap); err != nil {
		if k8sErrors.IsNotFound(err) {
			GinkgoLogr.Info("configmap not found")
			return
		}
		GinkgoLogr.Error(err, "error getting configmap")
		return
	}
	var config gitops.Config
	if err := yaml.Unmarshal([]byte(configMap.Data[gitopsConfigmapDataKey]), &config); err != nil {
		GinkgoLogr.Error(err, "error unmarshalling configmap data")
		return
	}
	GinkgoLogr.Info("configmap data", "config", config)
}

func updateGitopsConfig(ctx context.Context, updateFn func(config gitops.Config) gitops.Config) error {
	exists := true
	var configMap v1.ConfigMap
	var config gitops.Config
	if err := k8sClient.Get(ctx, ctrlClient.ObjectKey{Namespace: namespace, Name: gitopsConfigmapName}, &configMap); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return err
		}
		exists = false
		configMap = v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      gitopsConfigmapName,
			},
			Data: map[string]string{},
		}
	} else {
		if err := yaml.Unmarshal([]byte(configMap.Data[gitopsConfigmapDataKey]), &config); err != nil {
			return err
		}
	}

	updated := updateFn(config)
	updatedYaml, err := yaml.Marshal(updated)
	if err != nil {
		return err
	}
	configMap.Data[gitopsConfigmapDataKey] = string(updatedYaml)
	if exists {
		return k8sClient.Update(ctx, &configMap)
	} else {
		return k8sClient.Create(ctx, &configMap)
	}

}

func operatorConfigForVersion(version string) OperatorConfig {
	return OperatorConfig{
		DeploymentName:       getDeploymentName(version),
		Image:                fmt.Sprintf("quay.io/rhacs-eng/stackrox-operator:%s", version),
		CentralLabelSelector: fmt.Sprintf("rhacs.redhat.com/version-selector=%s", version),
	}
}

func getDeploymentName(version string) string {
	return fmt.Sprintf("rhacs-operator-e2e-%s", version)
}

func getOperatorDeployments(ctx context.Context) ([]appsv1.Deployment, error) {
	deployments := appsv1.DeploymentList{}
	labels := map[string]string{"app.kubernetes.io/name": "rhacs-operator"}
	err := k8sClient.List(ctx, &deployments,
		ctrlClient.InNamespace("rhacs"),
		ctrlClient.MatchingLabels(labels))
	if err != nil {
		return nil, err
	}
	return deployments.Items, nil
}

func getDeployment(ctx context.Context, namespace string, name string) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	err := k8sClient.Get(ctx, ctrlClient.ObjectKey{Namespace: namespace, Name: name}, deployment)
	return deployment, err
}

func getVPA(ctx context.Context, namespace string, name string) (*verticalpodautoscalingv1.VerticalPodAutoscaler, error) {
	autoscaler := &verticalpodautoscalingv1.VerticalPodAutoscaler{}
	err := k8sClient.Get(ctx, ctrlClient.ObjectKey{Namespace: namespace, Name: name}, autoscaler)
	return autoscaler, err
}

func getContainer(name string, containers []v1.Container) (*v1.Container, error) {
	for _, container := range containers {
		if container.Name == name {
			return &container, nil
		}
	}
	return nil, fmt.Errorf("container %s not found", name)
}

type deploymentAssertion func(deploy *appsv1.Deployment) error

var operatorHasImageAssertion = func(image string) deploymentAssertion {
	return func(deploy *appsv1.Deployment) error {
		container, err := getContainer("manager", deploy.Spec.Template.Spec.Containers)
		if err != nil {
			return err
		}
		if container.Image != image {
			return fmt.Errorf("incorrect image %s", container.Image)
		}
		return nil
	}
}

var operatorHasCentralLabelSelectorAssertion = func(labelSelector string) deploymentAssertion {
	return func(deploy *appsv1.Deployment) error {
		container, err := getContainer("manager", deploy.Spec.Template.Spec.Containers)
		if err != nil {
			return err
		}
		for _, envVar := range container.Env {
			if envVar.Name == "CENTRAL_LABEL_SELECTOR" {
				if envVar.Value != labelSelector {
					return fmt.Errorf("incorrect value %s", envVar.Value)
				}
			}
		}
		return nil
	}
}

func validateDeployment(deployment *appsv1.Deployment, assertions ...deploymentAssertion) error {
	for _, assertion := range assertions {
		err := assertion(deployment)
		if err != nil {
			return err
		}
	}
	return nil
}

type OperatorConfig struct {
	DeploymentName       string
	Image                string
	CentralLabelSelector string
}

func operatorMatchesConfig(ctx context.Context, config OperatorConfig) func() error {
	return func() error {
		deploy, err := getDeployment(ctx, "rhacs", config.DeploymentName)
		if err != nil {
			println("Got err", err.Error(), config.DeploymentName)
			return err
		}
		return validateDeployment(deploy,
			operatorHasImageAssertion(config.Image),
			operatorHasCentralLabelSelectorAssertion(config.CentralLabelSelector),
		)
	}
}

func getLabelAndVersionFromOperatorConfig(operatorConfig OperatorConfig) (string, string, error) {
	selector := operatorConfig.CentralLabelSelector
	selectorParts := strings.Split(selector, "=")
	if len(selectorParts) != 2 {
		return "", "", fmt.Errorf("invalid selector %s", selector)
	}
	versionLabelKey := selectorParts[0]
	versionLabelValue := selectorParts[1]
	return versionLabelKey, versionLabelValue, nil
}

func overrideAllCentralsToBeReconciledByOperator(operatorConfig OperatorConfig) gitops.TenantResourceOverride {
	return overrideAllCentralsWithPatch(reconciledByOperatorPatch(operatorConfig))
}

func overrideAllCentralsToUseMinimalResources() gitops.TenantResourceOverride {
	return overrideAllCentralsWithPatch(minimalCentralResourcesPatch())
}

func overrideAllCentralsWithPatch(patch string) gitops.TenantResourceOverride {
	return gitops.TenantResourceOverride{
		InstanceIDs: []string{"*"},
		Values:      patch,
	}
}

func overrideCentralWithPatch(centralID, patch string) gitops.TenantResourceOverride {
	return gitops.TenantResourceOverride{
		InstanceIDs: []string{centralID},
		Values:      patch,
	}
}

func reconciledByOperatorPatch(operatorConfig OperatorConfig) string {
	_, value, err := getLabelAndVersionFromOperatorConfig(operatorConfig)
	if err != nil {
		panic(err)
	}
	return `rolloutGroup: ` + value
}

func minimalCentralResourcesPatch() string {
	return `
spec:
  monitoring:
    openshift:
      enabled: false
  network:
    policies: Disabled
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
          cpu: 10000m
          memory: 20Gi
        requests:
          cpu: 100m
          memory: 100Mi
`
}

func defaultTenantResourceValues() string {
	return `
centralVpaEnabled: false
rolloutGroup: dev
centralResources:
  limits:
    cpu: null
    memory: 1Gi
  requests:
    cpu: 100m
    memory: 100Mi
centralDbResources:
  limits:
    cpu: null
    memory: 1Gi
  requests:
    cpu: 100m
    memory: 100Mi
scannerResources:
  limits:
    cpu: null
    memory: 2Gi
  requests:
    cpu: 100m
    memory: 100Mi
scannerDbResources:
  limits:
    cpu: null
    memory: 3Gi
  requests:
    cpu: 100m
    memory: 100Mi
`
}

func tenantResourcesWithCentralVpaEnabled() string {
	return `centralVpaEnabled: true`
}

func mustJson(obj interface{}) []byte {
	b, err := json.Marshal(obj)
	Expect(err).ToNot(HaveOccurred())
	return b
}

func defaultGitopsConfig() gitops.Config {
	return gitops.Config{
		TenantResources: defaultTenantResources(),
		Applications:    defaultApplications(),
	}
}

func defaultTenantResources() gitops.TenantResourceConfig {
	return gitops.TenantResourceConfig{
		Default: defaultTenantResourceValues(),
	}
}

func defaultApplications() []argocd.Application {
	return []argocd.Application{
		crdsApplication(),
		operatorsApplication(operator462()),
	}
}

func restoreDefaultGitopsConfig() error {
	return putGitopsConfig(context.Background(), defaultGitopsConfig())
}

func operator462() OperatorConfig {
	return OperatorConfig{
		DeploymentName:       "rhacs-operator-4.6.2",
		Image:                "registry.redhat.io/advanced-cluster-security/rhacs-rhel8-operator@sha256:a96572d0df791da60763dec4b4f0f52124772c3649303170968806dcc3de8269",
		CentralLabelSelector: "rhacs.redhat.com/version-selector=dev",
	}
}

func operatorsApplication(operators ...OperatorConfig) argocd.Application {
	images := make([]map[string]interface{}, len(operators))
	for i, operator := range operators {
		images[i] = map[string]interface{}{
			"deploymentName":                  operator.DeploymentName,
			"image":                           operator.Image,
			"centralLabelSelector":            operator.CentralLabelSelector,
			"securedClusterReconcilerEnabled": false,
		}
	}

	return argocd.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rhacs-operators",
		},
		Spec: argocd.ApplicationSpec{
			IgnoreDifferences: argocd.IgnoreDifferences{
				{
					Kind:         "ServiceAccount",
					JSONPointers: []string{"/imagePullSecrets"},
				},
			},
			Destination: argocd.ApplicationDestination{
				Namespace: "rhacs",
				Server:    "https://kubernetes.default.svc",
			},
			Project: "default",
			Source: &argocd.ApplicationSource{
				Helm: &argocd.ApplicationSourceHelm{
					ValuesObject: &runtime.RawExtension{
						Raw: mustJson(map[string]interface{}{
							"operator": map[string]interface{}{
								"images": images,
							},
						}),
					},
				},
				Path:           "rhacs-operator-legacy",
				RepoURL:        "https://github.com/stackrox/acscs-manifests",
				TargetRevision: "HEAD",
			},
			SyncPolicy: &argocd.SyncPolicy{
				Automated: &argocd.SyncPolicyAutomated{
					Prune:    true,
					SelfHeal: true,
				},
			},
		},
	}
}

func crdsApplication() argocd.Application {
	return argocd.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rhacs-crds",
		},
		Spec: argocd.ApplicationSpec{
			Destination: argocd.ApplicationDestination{
				Namespace: "rhacs",
				Server:    "https://kubernetes.default.svc",
			},
			Project: "default",
			Source: &argocd.ApplicationSource{
				Directory: &argocd.ApplicationSourceDirectory{
					Include: "{platform.stackrox.io_centrals.yaml,platform.stackrox.io_securedclusters.yaml}",
				},
				Path:           "operator/bundle/manifests",
				RepoURL:        "https://github.com/stackrox/stackrox",
				TargetRevision: "4.6.2",
			},
			SyncPolicy: &argocd.SyncPolicy{
				Automated: &argocd.SyncPolicyAutomated{
					Prune:    true,
					SelfHeal: true,
				},
				SyncOptions: argocd.SyncOptions{"ServerSideApply=true"},
			},
		},
	}
}
