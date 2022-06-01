package presenters

import (
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	v1 "github.com/stackrox/acs-fleet-manager/pkg/api/manageddinosaurs.manageddinosaur.mas/v1"
)

// TODO(create-ticket): implement configurable central and scanner resources
const defaultCentralRequestMemory = "1Gi"
const defaultCentralRequestCpu = "1000m"
const defaultCentralLimitMemory = "4Gi"
const defaultCentralLimitCpu = "1000m"

const defaultScannerAnalyzerRequestMemory = "500Mi"
const defaultScannerAnalyzerRequestCpu = "500m"
const defaultScannerAnalyzerLimitMemory = "2500Mi"
const defaultScannerAnalyzerLimitCpu = "2000m"

const defaultScannerAnalyzerAutoScaling = "enabled"
const defaultScannerAnalyzerScalingReplicas = 1
const defaultScannerAnalyzerScalingMinReplicas = 1
const defaultScannerAnalyzerScalingMaxReplicas = 3

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
			//TODO(create-ticket): set deletion timestamp in deletion process
			DeletionTimestamp: "TODO get deletion timestamp",
		},
		Spec: private.ManagedCentralAllOfSpec{
			Owners: from.Spec.Owners,
			Endpoint: private.ManagedCentralAllOfSpecEndpoint{
				Host: from.Spec.Endpoint.Host,
				Tls:  &private.ManagedCentralAllOfSpecEndpointTls{},
			},
			Versions: private.ManagedCentralVersions{
				Central:         from.Spec.Versions.Dinosaur,
				CentralOperator: from.Spec.Versions.DinosaurOperator,
			},
			// TODO(create-ticket): add additional cas to public create/get centrals api and internal models
			AdditionalCAs: []private.ManagedCentralAllOfSpecAdditionalCAs{},
			Central: private.ManagedCentralAllOfSpecCentral{
				Resources: private.Resources{
					Requests: private.ResourceReference{
						Cpu:    defaultCentralRequestCpu,
						Memory: defaultCentralRequestMemory,
					},
					Limits: private.ResourceReference{
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
					Resources: private.Resources{
						Requests: private.ResourceReference{
							Cpu:    defaultScannerAnalyzerRequestCpu,
							Memory: defaultScannerAnalyzerRequestMemory,
						},
						Limits: private.ResourceReference{
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
			Deleted: from.Spec.Deleted,
		},
	}
	return res
}
