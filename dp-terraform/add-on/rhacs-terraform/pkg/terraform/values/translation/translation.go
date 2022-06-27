package translation

import (
	"context"
	"fmt"

	"github.com/stackrox/acs-fleet-manager/dp-terraform/add-on/rhacs-terraform/api/v1alpha1"
	"github.com/stackrox/acs-fleet-manager/dp-terraform/add-on/rhacs-terraform/pkg/values/translation"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// Translator translates and enriches helm values
type Translator struct {
	chartPath string
}

func NewTranslator(chartPath string) Translator {
	return Translator{chartPath: chartPath}
}

// Translate translates and enriches helm values
func (t Translator) Translate(ctx context.Context, u *unstructured.Unstructured) (chartutil.Values, error) {
	valuesFilePath := fmt.Sprintf("%s/%s", t.chartPath, chartutil.ValuesfileName)
	defaultValues, err := chartutil.ReadValuesFile(valuesFilePath)
	if err != nil {
		return nil, err
	}

	// NOTE: this should check the kind if we eventually support more CRDs
	terraform := v1alpha1.Terraform{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &terraform)
	if err != nil {
		return nil, err
	}

	valsFromCR, err := translate(terraform)
	if err != nil {
		return nil, err
	}

	values := chartutil.CoalesceTables(valsFromCR, defaultValues)
	return values, nil
}


// translate translates a Terraform CR into helm values.
// This should only set values for fields that are not set to a zero value.
func translate(t v1alpha1.Terraform) (chartutil.Values, error) {
	v := translation.NewValuesBuilder()

	fleetshardSync := translation.NewValuesBuilder()
	// Not checking for zero value for fields declaraded as mandatory in config/crd
	fleetshardSync.SetString("ocmToken", &t.Spec.FleetshardSync.OcmToken)
	fleetshardSync.SetString("fleetManagerEndpoint", &t.Spec.FleetshardSync.FleetManagerEndpoint)
	fleetshardSync.SetString("clusterId", &t.Spec.FleetshardSync.ClusterId)
	redHatSSO := translation.NewValuesBuilder()
	redHatSSO.SetString("clientId", &t.Spec.FleetshardSync.RedHatSSO.ClientId)
	redHatSSO.SetString("clientSecret", &t.Spec.FleetshardSync.RedHatSSO.ClientSecret)
	fleetshardSync.AddChild("redHatSSO", &redHatSSO)
	v.AddChild("fleetshardSync", &fleetshardSync)

	acsOperator := translation.NewValuesBuilder()
	if t.Spec.AcsOperator.Enabled {
		acsOperator.SetBool("enabled", &t.Spec.AcsOperator.Enabled)	
	}
	if t.Spec.AcsOperator.StartingCSV != "" {
		acsOperator.SetString("startingCSV", &t.Spec.AcsOperator.StartingCSV)
	}
	v.AddChild("acsOperator", &acsOperator)
	
	observability := translation.NewValuesBuilder()
	if t.Spec.Observability.Enabled {
		observability.SetBool("enabled", &t.Spec.Observability.Enabled)
	}
	// TODO(create-ticket): validate fields that should be mandatory if obs is enabled
	github := translation.NewValuesBuilder()
	github.SetString("accessToken", &t.Spec.Observability.Github.AccessToken)
	if t.Spec.Observability.Github.Repository != "" {
		github.SetString("repository", &t.Spec.Observability.Github.Repository)
	}
	observability.AddChild("github", &github)
	observatorium := translation.NewValuesBuilder()
	if t.Spec.Observability.Observatorium.Gateway != "" {
		observatorium.SetString("gateway", &t.Spec.Observability.Observatorium.Gateway)
	}
	observatorium.SetString("metricsClientId", &t.Spec.Observability.Observatorium.MetricsClientId)
	observatorium.SetString("metricsSecret", &t.Spec.Observability.Observatorium.MetricsSecret)
	observability.AddChild("observatorium", &observatorium)
	v.AddChild("observability", &observability)

	return v.Build()
}