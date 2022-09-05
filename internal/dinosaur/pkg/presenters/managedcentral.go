package presenters

import (
	"encoding/json"
	"time"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/defaults"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
				ClientId:     c.centralConfig.RhSsoClientID,
				OwnerOrgId:   from.OrganisationID,
				OwnerUserId:  from.OwnerUserID,
				Issuer:       c.centralConfig.RhSsoIssuer,
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
						corev1.ResourceCPU.String():    orDefaultQty(central.Resources.Requests[corev1.ResourceCPU], defaults.CentralRequestCPU).String(),
						corev1.ResourceMemory.String(): orDefaultQty(central.Resources.Requests[corev1.ResourceMemory], defaults.CentralRequestMemory).String(),
					},
					Limits: map[string]string{
						corev1.ResourceCPU.String():    orDefaultQty(central.Resources.Limits[corev1.ResourceCPU], defaults.CentralLimitCPU).String(),
						corev1.ResourceMemory.String(): orDefaultQty(central.Resources.Limits[corev1.ResourceMemory], defaults.CentralLimitMemory).String(),
					},
				},
			},
			Scanner: private.ManagedCentralAllOfSpecScanner{
				Analyzer: private.ManagedCentralAllOfSpecScannerAnalyzer{
					Scaling: private.ManagedCentralAllOfSpecScannerAnalyzerScaling{
						AutoScaling: orDefaultString(scanner.Analyzer.Scaling.AutoScaling, defaults.ScannerAnalyzerAutoScaling),
						Replicas:    orDefaultInt32(scanner.Analyzer.Scaling.Replicas, defaults.ScannerAnalyzerScalingReplicas),
						MinReplicas: orDefaultInt32(scanner.Analyzer.Scaling.MinReplicas, defaults.ScannerAnalyzerScalingMinReplicas),
						MaxReplicas: orDefaultInt32(scanner.Analyzer.Scaling.MaxReplicas, defaults.ScannerAnalyzerScalingMaxReplicas),
					},
					Resources: private.ResourceRequirements{
						Requests: map[string]string{
							corev1.ResourceCPU.String():    orDefaultQty(scanner.Analyzer.Resources.Requests[corev1.ResourceCPU], defaults.ScannerAnalyzerRequestCPU).String(),
							corev1.ResourceMemory.String(): orDefaultQty(scanner.Analyzer.Resources.Requests[corev1.ResourceMemory], defaults.ScannerAnalyzerRequestMemory).String(),
						},
						Limits: map[string]string{
							corev1.ResourceCPU.String():    orDefaultQty(scanner.Analyzer.Resources.Limits[corev1.ResourceCPU], defaults.ScannerAnalyzerLimitCPU).String(),
							corev1.ResourceMemory.String(): orDefaultQty(scanner.Analyzer.Resources.Limits[corev1.ResourceMemory], defaults.ScannerAnalyzerLimitMemory).String(),
						},
					},
				},
				Db: private.ManagedCentralAllOfSpecScannerDb{
					// TODO:(create-ticket): add DB configuration values to ManagedCentral Scanner
					Host: "dbhost.rhacs-psql-instance",
					Resources: private.ResourceRequirements{
						Requests: map[string]string{
							corev1.ResourceCPU.String():    orDefaultQty(scanner.Db.Resources.Requests[corev1.ResourceCPU], defaults.ScannerDbRequestCPU).String(),
							corev1.ResourceMemory.String(): orDefaultQty(scanner.Db.Resources.Requests[corev1.ResourceMemory], defaults.ScannerDbRequestMemory).String(),
						},
						Limits: map[string]string{
							corev1.ResourceCPU.String():    orDefaultQty(scanner.Db.Resources.Limits[corev1.ResourceCPU], defaults.ScannerDbLimitCPU).String(),
							corev1.ResourceMemory.String(): orDefaultQty(scanner.Db.Resources.Limits[corev1.ResourceMemory], defaults.ScannerDbLimitMemory).String(),
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
