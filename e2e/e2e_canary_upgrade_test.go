package e2e

import (
	"context"
	"fmt"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
	"strings"
)

const (
	namespace              = "acscs"
	gitopsConfigmapName    = "fleet-manager-gitops-config"
	gitopsConfigmapDataKey = "config.yaml"
	operatorVersion1       = "4.2.2-rc.0"
	operatorVersion2       = "4.2.0-366-g069902f3f9"
	centralDeploymentName  = "central"
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
		var gitops gitops.Config

		It("get gitops configmap", func() {
			gitops, err = getGitopsConfig(ctx)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should deploy operator 1"+operatorConfig1.GetDeploymentName(), func() {
			// update gitops config to install one operator
			gitops.RHACSOperators.Configs = []operator.OperatorConfig{operatorConfig1}
			err = updateGitopsConfig(ctx, gitops)
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
			gitops.RHACSOperators.Configs = []operator.OperatorConfig{operatorConfig1, operatorConfig2}
			err = updateGitopsConfig(ctx, gitops)
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
			gitops.RHACSOperators.Configs = []operator.OperatorConfig{operatorConfig1}
			err = updateGitopsConfig(ctx, gitops)
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

		It("should deploy operator 2 with label selector "+operatorConfig2.GetCentralLabelSelector(), func() {
			Eventually(operatorMatchesConfig(ctx, operatorConfig2)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})
	})

	Describe("should upgrade the central", func() {
		ctx := context.Background()
		var createdCentral *public.CentralRequest
		var centralNamespace string
		var gitops gitops.Config
		operatorConfig1 := operatorConfigForVersion(operatorVersion1)
		operatorConfig2 := operatorConfigForVersion(operatorVersion2)

		gitops, err = getGitopsConfig(ctx)
		Expect(err).ToNot(HaveOccurred())

		It("run only one operator with version: "+operatorVersion1, func() {
			gitops.RHACSOperators.Configs = []operator.OperatorConfig{operatorConfig1}
			err = updateGitopsConfig(ctx, gitops)
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
			gitops.RHACSOperators.Configs = []operator.OperatorConfig{operatorConfig1, operatorConfig2}
			patch := `
metadata:
  labels:
    rhacs.redhat.com/version-selector: "` + operatorVersion2 + `"
spec:
  central:
    monitoring:
      openshift:
        enabled: false
    resources:
      limits:
        cpu: null
        memory: 1Gi
      requests:
        cpu: 100m
        memory: 200Mi
  scanner:
    analyzer:
      resources:
        limits:
          cpu: null
          memory: 1Gi
        requests:
          cpu: 100m
          memory: 500Mi
    db:
      resources:
        limits:
          cpu: null
          memory: 1Gi
        requests:
          cpu: 100m
          memory: 500Mi
`
			gitops.Centrals.Overrides[0].Patch = patch
			err = updateGitopsConfig(ctx, gitops)
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

func updateGitopsConfig(ctx context.Context, config gitops.Config) error {
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

	err = k8sClient.Update(ctx, configMap)
	if err != nil {
		return err
	}

	return nil
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
