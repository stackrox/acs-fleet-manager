package presenters

import (
	"time"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	v1 "github.com/stackrox/acs-fleet-manager/pkg/api/manageddinosaurs.manageddinosaur.mas/v1"
)

// TODO(create-ticket): implement configurable central and scanner resources
const (
	defaultCentralRequestMemory = "250Mi"
	defaultCentralRequestCpu    = "250m"
	defaultCentralLimitMemory   = "4Gi"
	defaultCentralLimitCpu      = "1000m"

	defaultScannerAnalyzerRequestMemory = "100Mi"
	defaultScannerAnalyzerRequestCpu    = "250m"
	defaultScannerAnalyzerLimitMemory   = "2500Mi"
	defaultScannerAnalyzerLimitCpu      = "2000m"

	defaultScannerAnalyzerAutoScaling        = "enabled"
	defaultScannerAnalyzerScalingReplicas    = 1
	defaultScannerAnalyzerScalingMinReplicas = 1
	defaultScannerAnalyzerScalingMaxReplicas = 3
)

func PresentManagedDinosaur(from *v1.ManagedDinosaur) private.ManagedCentral {
	res := private.ManagedCentral{
		Id:   from.Annotations["mas/id"],
		Kind: from.Kind,
		Metadata: private.ManagedCentralAllOfMetadata{
			Name:      from.Name,
			Namespace: from.Namespace,
			Annotations: private.ManagedCentralAllOfMetadataAnnotations{
				MasId:          from.Annotations["mas/id"],
				MasPlacementId: from.Annotations["mas/placementId"],
			},
		},
		Spec: private.ManagedCentralAllOfSpec{
			Owners: from.Spec.Owners,
			Endpoint: private.ManagedCentralAllOfSpecEndpoint{
				Host: from.Spec.Endpoint.Host,
				Tls: &private.ManagedCentralAllOfSpecEndpointTls{
					Cert: "cert-data",
					Key:  "key-data",
				},
			},
			Versions: private.ManagedCentralVersions{
				Central:         from.Spec.Versions.Dinosaur,
				CentralOperator: from.Spec.Versions.DinosaurOperator,
			},
			// TODO(create-ticket): add additional CAs to public create/get centrals api and internal models
			Central: private.ManagedCentralAllOfSpecCentral{
				Resources: private.ResourceRequirements{
					Requests: private.ResourceList{
						Cpu:    defaultCentralRequestCpu,
						Memory: defaultCentralRequestMemory,
					},
					Limits: private.ResourceList{
						Cpu:    defaultCentralLimitCpu,
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
							Cpu:    defaultScannerAnalyzerRequestCpu,
							Memory: defaultScannerAnalyzerRequestMemory,
						},
						Limits: private.ResourceList{
							Cpu:    defaultScannerAnalyzerLimitCpu,
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
	}

	if from.DeletionTimestamp != nil {
		res.Metadata.DeletionTimestamp = from.DeletionTimestamp.Format(time.RFC3339)
	}

	return res
}
