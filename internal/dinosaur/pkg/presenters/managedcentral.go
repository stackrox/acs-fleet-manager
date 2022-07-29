package presenters

import (
	"time"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
)

// TODO(create-ticket): implement configurable central and scanner resources
const (
	defaultCentralRequestMemory = "250Mi"
	defaultCentralRequestCPU    = "250m"
	defaultCentralLimitMemory   = "4Gi"
	defaultCentralLimitCPU      = "1000m"

	defaultScannerAnalyzerRequestMemory = "100Mi"
	defaultScannerAnalyzerRequestCPU    = "250m"
	defaultScannerAnalyzerLimitMemory   = "2500Mi"
	defaultScannerAnalyzerLimitCPU      = "2000m"

	defaultScannerAnalyzerAutoScaling        = "enabled"
	defaultScannerAnalyzerScalingReplicas    = 1
	defaultScannerAnalyzerScalingMinReplicas = 1
	defaultScannerAnalyzerScalingMaxReplicas = 3
)

// ManagedCentralPresenter helper service which converts Central DB representation to the private API representation
type ManagedCentralPresenter struct {
	centralConfig *config.DinosaurConfig
}

// NewManagedCentralPresenter creates a new instance of ManagedCentralPresenter
func NewManagedCentralPresenter(config *config.DinosaurConfig) *ManagedCentralPresenter {
	return &ManagedCentralPresenter{centralConfig: config}
}

// PresentManagedCentral converts DB representation of Central to the private API representation
func (c *ManagedCentralPresenter) PresentManagedCentral(from *dbapi.CentralRequest) private.ManagedCentral {
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
				// TODO: ROX-11593: make part of centralConfig
				ClientId:    "rhacs-ms-dev",
				OwnerOrgId:  from.OrganisationID,
				OwnerUserId: from.OwnerUserID,
			},
			UiEndpoint: private.ManagedCentralAllOfSpecUiEndpoint{
				Host: from.Host,
				Tls: private.ManagedCentralAllOfSpecUiEndpointTls{
					Cert: c.centralConfig.DinosaurTLSCert,
					Key:  c.centralConfig.DinosaurTLSKey,
				},
			},
			Versions: private.ManagedCentralVersions{
				Central:         from.DesiredCentralVersion,
				CentralOperator: from.DesiredCentralOperatorVersion,
			},
			// TODO(create-ticket): add additional CAs to public create/get centrals api and internal models
			Central: private.ManagedCentralAllOfSpecCentral{
				Resources: private.ResourceRequirements{
					Requests: private.ResourceList{
						Cpu:    defaultCentralRequestCPU,
						Memory: defaultCentralRequestMemory,
					},
					Limits: private.ResourceList{
						Cpu:    defaultCentralLimitCPU,
						Memory: defaultCentralLimitMemory,
					},
				},
			},
			Scanner: private.ManagedCentralAllOfSpecScanner{
				Analyzer: private.ManagedCentralAllOfSpecScannerAnalyzer{
					Scaling: private.ManagedCentralAllOfSpecScannerAnalyzerScaling{
						AutoScaling: defaultScannerAnalyzerAutoScaling,
						Replicas:    defaultScannerAnalyzerScalingReplicas,
						MinReplicas: defaultScannerAnalyzerScalingMinReplicas,
						MaxReplicas: defaultScannerAnalyzerScalingMaxReplicas,
					},
					Resources: private.ResourceRequirements{
						Requests: private.ResourceList{
							Cpu:    defaultScannerAnalyzerRequestCPU,
							Memory: defaultScannerAnalyzerRequestMemory,
						},
						Limits: private.ResourceList{
							Cpu:    defaultScannerAnalyzerLimitCPU,
							Memory: defaultScannerAnalyzerLimitMemory,
						},
					},
				},
				Db: private.ManagedCentralAllOfSpecScannerDb{
					// TODO:(create-ticket): add DB configuration values to ManagedCentral Scanner
					Host: "dbhost.rhacs-psql-instance",
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
