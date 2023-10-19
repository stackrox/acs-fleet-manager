package e2e

import (
	"context"
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/operator"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/gitops"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/pkg/features"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"golang.org/x/exp/slices"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	namespace              = "acscs"
	gitopsConfigmapName    = "fleet-manager-gitops-config"
	gitopsConfigmapDataKey = "config.yaml"
	operatorVersion1       = "4.2.2-rc.0"
	operatorVersion2       = "4.2.0-366-g069902f3f9"
	centralDeploymentName  = "central"
)

var (
	crdUrls = []string{
		"https://raw.githubusercontent.com/stackrox/stackrox/4.2.1/operator/bundle/manifests/platform.stackrox.io_securedclusters.yaml",
		"https://raw.githubusercontent.com/stackrox/stackrox/4.2.1/operator/bundle/manifests/platform.stackrox.io_centrals.yaml",
	}
)

var _ = Describe("Fleetshard-sync Targeted Upgrade", func() {
	var client *fleetmanager.Client
	var err error

	BeforeEach(func() {
		if !features.TargetedOperatorUpgrades.Enabled() || !runCanaryUpgradeTests {
			Skip("Skipping canary upgrade test")
		}
		option := fleetmanager.OptionFromEnv()
		auth, err := fleetmanager.NewStaticAuth(fleetmanager.StaticOption{StaticToken: option.Static.StaticToken})
		Expect(err).ToNot(HaveOccurred())
		client, err = fleetmanager.NewClient(fleetManagerEndpoint, auth)
		Expect(err).ToNot(HaveOccurred())
	})

	if fmEndpointEnv := os.Getenv("FLEET_MANAGER_ENDPOINT"); fmEndpointEnv != "" {
		fleetManagerEndpoint = fmEndpointEnv
	}

	operatorConfig1 := operatorConfigForVersion(operatorVersion1)
	operatorConfig2 := operatorConfigForVersion(operatorVersion2)

	Describe("should run ACS operators", func() {
		ctx := context.Background()

		It("should deploy operator 1 "+operatorConfig1.GetDeploymentName(), func() {
			// update gitops config to install one operator
			err = putGitopsConfig(ctx, gitops.Config{
				RHACSOperators: operator.OperatorConfigs{
					CRDURLs: crdUrls,
					Configs: []operator.OperatorConfig{operatorConfig1},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			Eventually(expectNumberOfOperatorDeployments(ctx, 1, getDeploymentName(operatorVersion1))).WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should run operator 1 with central label selector "+operatorConfig1.GetCentralLabelSelector(), func() {
			Eventually(operatorMatchesConfig(ctx, operatorConfig1)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should deploy two operators in different versions", func() {
			// add a second operator version to the gitops config
			err = putGitopsConfig(ctx, gitops.Config{
				RHACSOperators: operator.OperatorConfigs{
					CRDURLs: crdUrls,
					Configs: []operator.OperatorConfig{operatorConfig1, operatorConfig2},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			Eventually(expectNumberOfOperatorDeployments(ctx, 2, getDeploymentName(operatorVersion1), getDeploymentName(operatorVersion2))).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should deploy operator 1 with label selector "+operatorConfig1.GetCentralLabelSelector(), func() {
			Eventually(operatorMatchesConfig(ctx, operatorConfig1)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should deploy operator 2 with label selector "+operatorConfig2.GetCentralLabelSelector(), func() {
			Eventually(operatorMatchesConfig(ctx, operatorConfig2)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should delete operator 2 and only run operator 1", func() {
			err = putGitopsConfig(ctx, gitops.Config{
				RHACSOperators: operator.OperatorConfigs{
					CRDURLs: crdUrls,
					Configs: []operator.OperatorConfig{operatorConfig1},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			Eventually(expectNumberOfOperatorDeployments(ctx, 1, getDeploymentName(operatorVersion1))).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should deploy operator 1 with label selector "+operatorConfig1.GetCentralLabelSelector(), func() {
			Eventually(operatorMatchesConfig(ctx, operatorConfig1)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})
	})

	Describe("should upgrade the central", func() {
		ctx := context.Background()
		var createdCentral *public.CentralRequest
		var centralNamespace string
		operatorConfig1 := operatorConfigForVersion(operatorVersion1)
		operatorConfig2 := operatorConfigForVersion(operatorVersion2)

		It("run only one operator with version: "+operatorVersion1, func() {
			err = putGitopsConfig(ctx, gitops.Config{
				RHACSOperators: operator.OperatorConfigs{
					CRDURLs: crdUrls,
					Configs: []operator.OperatorConfig{operatorConfig1},
				},
				Centrals: gitops.CentralsConfig{
					Overrides: []gitops.CentralOverride{
						overrideAllCentralsToBeReconciledByOperator(operatorConfig1),
						overrideAllCentralsToUseMinimalResources(),
					},
				},
			})
			Expect(err).To(BeNil())
			Eventually(expectNumberOfOperatorDeployments(ctx, 1, getDeploymentName(operatorVersion1))).
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
			Expect(err).To(BeNil())
			createdCentral = &resp
			Expect(err).To(BeNil())
			Expect(constants.CentralRequestStatusAccepted.String()).To(Equal(createdCentral.Status))
			centralNamespace, err = services.FormatNamespace(createdCentral.Id)
			Eventually(func() error {
				centralCR, err := getCentralCR(ctx, createdCentral.Name, centralNamespace)
				if err != nil {
					return fmt.Errorf("failed finding central CR: %v", err)
				}

				if centralCR.GetLabels()["rhacs.redhat.com/version-selector"] != operatorVersion1 {
					return fmt.Errorf("central CR does not have %s version-selector", operatorVersion1)
				}
				return nil
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Succeed())
			Eventually(func() error {
				centralDeployment, err := getDeployment(ctx, centralNamespace, centralDeploymentName)
				if err != nil {
					return fmt.Errorf("failed finding central deployment: %v", err)
				}
				if centralDeployment.Spec.Template.Spec.Containers[0].Image != "quay.io/rhacs-eng/main:"+operatorVersion1 {
					return fmt.Errorf("there is no central deployment with %s image tag", operatorVersion1)
				}
				return nil
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Succeed())
		})

		It("upgrade central", func() {
			Skip("Re-enable once https://github.com/stackrox/stackrox/pull/8156 is released with ACS/StackRox 4.3")
			err = putGitopsConfig(ctx, gitops.Config{
				RHACSOperators: operator.OperatorConfigs{
					CRDURLs: crdUrls,
					Configs: []operator.OperatorConfig{operatorConfig1, operatorConfig2},
				},
				Centrals: gitops.CentralsConfig{
					Overrides: []gitops.CentralOverride{
						overrideAllCentralsToBeReconciledByOperator(operatorConfig2),
						overrideAllCentralsToUseMinimalResources(),
					},
				},
			})
			Expect(err).To(BeNil())

			Eventually(func() error {
				centralDeployment, err := getDeployment(ctx, centralNamespace, centralDeploymentName)
				if err != nil {
					return fmt.Errorf("failed finding central deployment: %v", err)
				}
				if centralDeployment.Spec.Template.Spec.Containers[0].Image != "quay.io/rhacs-eng/main:"+operatorVersion2 {
					return fmt.Errorf("there is no central deployment with %s image tag", operatorVersion2)
				}
				return nil
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Succeed())
		})
	})

})

func expectNumberOfOperatorDeployments(ctx context.Context, n int, expectedDeploymentNames ...string) func() error {
	return func() error {
		var err error
		deployments, err := getOperatorDeployments(ctx)
		if err != nil {
			return err
		}
		if len(deployments) != n {
			err = fmt.Errorf("expected %d operator deployment, got %d", n, len(deployments))
		}

		found := false
		var names []string
		for _, deployment := range deployments {
			if slices.Contains(expectedDeploymentNames, deployment.GetName()) {
				found = true
				continue
			}
			names = append(names, deployment.GetName())
		}

		if !found {
			return fmt.Errorf("Expected deployments %s not found. Got '%s'. %w", expectedDeploymentNames, strings.Join(names, ","), err)
		}

		return err
	}
}

func getGitopsConfig(ctx context.Context) (gitops.Config, error) {
	var gitopsConfig gitops.Config
	configmap := &v1.ConfigMap{}

	err := k8sClient.Get(ctx, ctrlClient.ObjectKey{Namespace: namespace, Name: gitopsConfigmapName}, configmap)
	if err != nil {
		return gitops.Config{}, err
	}

	configmapData := []byte(configmap.Data[gitopsConfigmapDataKey])
	if err := yaml.Unmarshal(configmapData, &gitopsConfig); err != nil {
		return gitops.Config{}, err
	}

	return gitopsConfig, nil
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
		if !errors2.IsNotFound(err) {
			return err
		}
	} else {
		return nil
	}

	return k8sClient.Create(ctx, configMap)
}

func operatorConfigForVersion(version string) operator.OperatorConfig {
	return operator.OperatorConfig{
		"deploymentName":       getDeploymentName(version),
		"image":                fmt.Sprintf("quay.io/rhacs-eng/stackrox-operator:%s", version),
		"centralLabelSelector": fmt.Sprintf("rhacs.redhat.com/version-selector=%s", version),
	}
}

func getDeploymentName(version string) string {
	return fmt.Sprintf("rhacs-operator-e2e-%s", version)
}

func getOperatorDeployments(ctx context.Context) ([]appsv1.Deployment, error) {
	deployments := appsv1.DeploymentList{}
	labels := map[string]string{"app": "rhacs-operator"}
	err := k8sClient.List(ctx, &deployments,
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
	return nil, fmt.Errorf("container %s not found", name)
}

type operatorAssertion func(deploy *appsv1.Deployment) error

var operatorHasImage = func(image string) operatorAssertion {
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

var operatorHasCentralLabelSelector = func(labelSelector string) operatorAssertion {
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

func validateOperatorDeployment(deployment *appsv1.Deployment, assertions ...operatorAssertion) error {
	for _, assertion := range assertions {
		err := assertion(deployment)
		if err != nil {
			return err
		}
	}
	return nil
}

func operatorMatchesConfig(ctx context.Context, config operator.OperatorConfig) func() error {
	return func() error {
		deploy, err := getDeployment(ctx, operator.ACSOperatorNamespace, config.GetDeploymentName())
		if err != nil {
			println("Got err", err.Error(), config.GetDeploymentName())
			return err
		}

		return validateOperatorDeployment(deploy,
			operatorHasImage(config.GetImage()),
			operatorHasCentralLabelSelector(config.GetCentralLabelSelector()),
		)
	}
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

func overrideCentralWithPatch(centralID, patch string) gitops.CentralOverride {
	return gitops.CentralOverride{
		InstanceIDs: []string{centralID},
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

func forceReconcilePatch() string {
	return centralLabelPatch("rhacs.redhat.com/force-reconcile", "true")
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

func defaultGitopsConfig() gitops.Config {
	return gitops.Config{
		RHACSOperators: operator.OperatorConfigs{
			CRDURLs: []string{
				"https://raw.githubusercontent.com/stackrox/stackrox/4.2.1/operator/bundle/manifests/platform.stackrox.io_securedclusters.yaml",
				"https://raw.githubusercontent.com/stackrox/stackrox/4.2.1/operator/bundle/manifests/platform.stackrox.io_centrals.yaml",
			},
			Configs: []operator.OperatorConfig{
				{
					"deploymentName":                  "rhacs-operator-4.2.2-rc.0",
					"image":                           "quay.io/rhacs-eng/stackrox-operator:4.2.2-rc.0",
					"centralLabelSelector":            "rhacs.redhat.com/version-selector=4.2.2-rc.0",
					"securedClusterReconcilerEnabled": false,
				},
			},
		},
		Centrals: gitops.CentralsConfig{
			Overrides: []gitops.CentralOverride{
				{
					InstanceIDs: []string{"*"},
					Patch: `
metadata:
  labels:
    rhacs.redhat.com/version-selector: "4.2.2-rc.0"`,
				}, {
					InstanceIDs: []string{"*"},
					Patch: `
spec:
  monitoring:
    openshift:
      enabled: false
  central:
    db:
      resources:
        limits:
          cpu: null
          memory: 1Gi
        requests:
          cpu: 100m
          memory: 100Mi
    resources:
      limits:
        cpu: null
        memory: 1Gi
      requests:
        cpu: 100m
        memory: 100Mi
  scanner:
     analyzer:
       resources:
         limits:
           cpu: null
           memory: 2Gi
         requests:
           cpu: 100m
           memory: 100Mi
       scaling:
         autoScaling: "Disabled"
         replicas: 1
    db:
      resources:
        limits:
          cpu: null
          memory: 3Gi
        requests:
          cpu: 100m
          memory: 100Mi
`,
				},
			},
		},
	}
}

func restoreDefaultGitopsConfig() error {
	return putGitopsConfig(context.Background(), defaultGitopsConfig())
}
