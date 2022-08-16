package presenters

import (
	"encoding/json"
	"time"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	defaultCentralRequestMemory = resource.MustParse("250Mi")
	defaultCentralRequestCPU    = resource.MustParse("250m")
	defaultCentralLimitMemory   = resource.MustParse("4Gi")
	defaultCentralLimitCPU      = resource.MustParse("1000m")
	defaultCentralResources     = corev1.ResourceRequirements{
		Limits: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    defaultCentralLimitCPU,
			corev1.ResourceMemory: defaultCentralLimitMemory,
		},
		Requests: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    defaultCentralRequestCPU,
			corev1.ResourceMemory: defaultCentralRequestMemory,
		},
	}
	defaultScannerAnalyzerRequestMemory = resource.MustParse("100Mi")
	defaultScannerAnalyzerRequestCPU    = resource.MustParse("250m")
	defaultScannerAnalyzerLimitMemory   = resource.MustParse("2500Mi")
	defaultScannerAnalyzerLimitCPU      = resource.MustParse("2000m")
	defaultScannerAnalyzerResources     = corev1.ResourceRequirements{
		Limits: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    defaultScannerAnalyzerLimitCPU,
			corev1.ResourceMemory: defaultScannerAnalyzerLimitMemory,
		},
		Requests: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    defaultScannerAnalyzerRequestCPU,
			corev1.ResourceMemory: defaultScannerAnalyzerRequestMemory,
		},
	}

	defaultScannerAnalyzerAutoScaling              = "Enabled"
	defaultScannerAnalyzerScalingReplicas    int32 = 1
	defaultScannerAnalyzerScalingMinReplicas int32 = 1
	defaultScannerAnalyzerScalingMaxReplicas int32 = 3

	defaultScannerDbRequestMemory = resource.MustParse("100Mi")
	defaultScannerDbRequestCPU    = resource.MustParse("250m")
	defaultScannerDbLimitMemory   = resource.MustParse("2500Mi")
	defaultScannerDbLimitCPU      = resource.MustParse("2000m")
	defaultScannerDbResources     = corev1.ResourceRequirements{
		Limits: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    defaultScannerDbLimitCPU,
			corev1.ResourceMemory: defaultScannerDbLimitMemory,
		},
		Requests: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    defaultScannerDbRequestCPU,
			corev1.ResourceMemory: defaultScannerDbRequestMemory,
		},
	}
)

// ManagedCentralPresenter helper service which converts Central DB representation to the private API representation
type ManagedCentralPresenter struct {
	centralConfig *config.CentralConfig
}

// NewManagedCentralPresenter creates a new instance of ManagedCentralPresenter
func NewManagedCentralPresenter(config *config.CentralConfig) *ManagedCentralPresenter {
	return &ManagedCentralPresenter{centralConfig: config}
}

// PresentManagedCentral converts DB representation of Central to the private API representation
func (c *ManagedCentralPresenter) PresentManagedCentral(from *dbapi.CentralRequest) private.ManagedCentral {
	var central dbapi.CentralSpec
	var scanner dbapi.ScannerSpec

	if len(from.Central) > 0 {
		err := json.Unmarshal(from.Central, &central)
		if err != nil {
			// In case of a JSON unmarshaling problem we don't interrupt the complete workflow, instead we drop the resources
			// specification as a way of defensive programing.
			// TOOD: return error?
			glog.Errorf("Failed to unmarshal Central specification for Central request %q/%s: %v", from.Name, from.ClusterID, err)
			glog.Errorf("Ignoring Central specification for Central request %q/%s", from.Name, from.ClusterID)
		}
	}
	if len(from.Scanner) > 0 {
		err := json.Unmarshal(from.Scanner, &scanner)
		if err != nil {
			// In case of a JSON unmarshaling problem we don't interrupt the complete workflow, instead we drop the resources
			// specification as a way of defensive programing.
			// TOOD: return error?
			glog.Errorf("Failed to unmarshal Scanner specification for Central request %q/%s: %v", from.Name, from.ClusterID, err)
			glog.Errorf("Ignoring Scanner specification for Central request %q/%s", from.Name, from.ClusterID)
		}
	}

	res := private.ManagedCentral{
		Id:   from.ID,
		Kind: "ManagedCentral",
		Metadata: private.ManagedCentralAllOfMetadata{
			Name:      from.Name,
			Namespace: from.Namespace,
			Annotations: private.ManagedCentralAllOfMetadataAnnotations{
				MasId:          from.ID,
				MasPlacementId: from.PlacementID,
			},
		},
		Spec: private.ManagedCentralAllOfSpec{
			Owners: []string{
				from.Owner,
			},
			Auth: private.ManagedCentralAllOfSpecAuth{
				ClientSecret: c.centralConfig.RhSsoClientSecret, // pragma: allowlist secret
				// TODO(ROX-11593): make part of centralConfig
				ClientId:    "rhacs-ms-dev",
				OwnerOrgId:  from.OrganisationID,
				OwnerUserId: from.OwnerUserID,
				Issuer:      c.centralConfig.RhSsoIssuer,
			},
			UiEndpoint: private.ManagedCentralAllOfSpecUiEndpoint{
				Host: from.GetUIHost(),
				Tls: private.ManagedCentralAllOfSpecUiEndpointTls{
					Cert: c.centralConfig.CentralTLSCert,
					Key:  c.centralConfig.CentralTLSKey,
				},
			},
			DataEndpoint: private.ManagedCentralAllOfSpecDataEndpoint{
				Host: from.GetDataHost(),
			},
			Versions: private.ManagedCentralVersions{
				Central:         from.DesiredCentralVersion,
				CentralOperator: from.DesiredCentralOperatorVersion,
			},
			// TODO(create-ticket): add additional CAs to public create/get centrals api and internal models
			Central: private.ManagedCentralAllOfSpecCentral{
				Resources: private.ResourceRequirements{
					Requests: map[string]string{
						corev1.ResourceCPU.String():    orDefaultQty(central.Resources.Requests[corev1.ResourceCPU], defaultCentralRequestCPU).String(),
						corev1.ResourceMemory.String(): orDefaultQty(central.Resources.Requests[corev1.ResourceMemory], defaultCentralRequestMemory).String(),
					},
					Limits: map[string]string{
						corev1.ResourceCPU.String():    orDefaultQty(central.Resources.Limits[corev1.ResourceCPU], defaultCentralLimitCPU).String(),
						corev1.ResourceMemory.String(): orDefaultQty(central.Resources.Limits[corev1.ResourceMemory], defaultCentralLimitMemory).String(),
					},
				},
			},
			Scanner: private.ManagedCentralAllOfSpecScanner{
				Analyzer: private.ManagedCentralAllOfSpecScannerAnalyzer{
					Scaling: private.ManagedCentralAllOfSpecScannerAnalyzerScaling{
						AutoScaling: orDefaultString(scanner.Analyzer.Scaling.AutoScaling, defaultScannerAnalyzerAutoScaling),
						Replicas:    orDefaultInt32(scanner.Analyzer.Scaling.Replicas, defaultScannerAnalyzerScalingReplicas),
						MinReplicas: orDefaultInt32(scanner.Analyzer.Scaling.MinReplicas, defaultScannerAnalyzerScalingMinReplicas),
						MaxReplicas: orDefaultInt32(scanner.Analyzer.Scaling.MaxReplicas, defaultScannerAnalyzerScalingMaxReplicas),
					},
					Resources: private.ResourceRequirements{
						Requests: map[string]string{
							corev1.ResourceCPU.String():    orDefaultQty(scanner.Analyzer.Resources.Requests[corev1.ResourceCPU], defaultScannerAnalyzerRequestCPU).String(),
							corev1.ResourceMemory.String(): orDefaultQty(scanner.Analyzer.Resources.Requests[corev1.ResourceMemory], defaultScannerAnalyzerRequestMemory).String(),
						},
						Limits: map[string]string{
							corev1.ResourceCPU.String():    orDefaultQty(scanner.Analyzer.Resources.Limits[corev1.ResourceCPU], defaultScannerAnalyzerLimitCPU).String(),
							corev1.ResourceMemory.String(): orDefaultQty(scanner.Analyzer.Resources.Limits[corev1.ResourceMemory], defaultScannerAnalyzerLimitMemory).String(),
						},
					},
				},
				Db: private.ManagedCentralAllOfSpecScannerDb{
					// TODO:(create-ticket): add DB configuration values to ManagedCentral Scanner
					Host: "dbhost.rhacs-psql-instance",
					Resources: private.ResourceRequirements{
						Requests: map[string]string{
							corev1.ResourceCPU.String():    orDefaultQty(scanner.Db.Resources.Requests[corev1.ResourceCPU], defaultScannerDbRequestCPU).String(),
							corev1.ResourceMemory.String(): orDefaultQty(scanner.Db.Resources.Requests[corev1.ResourceMemory], defaultScannerDbRequestMemory).String(),
						},
						Limits: map[string]string{
							corev1.ResourceCPU.String():    orDefaultQty(scanner.Db.Resources.Limits[corev1.ResourceCPU], defaultScannerDbLimitCPU).String(),
							corev1.ResourceMemory.String(): orDefaultQty(scanner.Db.Resources.Limits[corev1.ResourceMemory], defaultScannerDbLimitMemory).String(),
						},
					},
				},
			},
		},
		RequestStatus: from.Status,
	}

	if from.DeletionTimestamp != nil {
		res.Metadata.DeletionTimestamp = from.DeletionTimestamp.Format(time.RFC3339)
	}

	return res
}

func orDefaultQty(qty resource.Quantity, def resource.Quantity) *resource.Quantity {
	if qty != (resource.Quantity{}) {
		return &qty
	}
	return &def
}

func orDefaultString(s string, def string) string {
	if s != "" {
		return s
	}
	return def
}

func orDefaultInt32(i int32, def int32) int32 {
	if i != 0 {
		return i
	}
	return def
}
