package presenters

import (
	"time"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	v1 "github.com/stackrox/acs-fleet-manager/pkg/api/manageddinosaurs.manageddinosaur.mas/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	defaultCentralRequestMemory = resource.MustParse("250Mi")
	defaultCentralRequestCPU    = resource.MustParse("250m")
	defaultCentralLimitMemory   = resource.MustParse("4Gi")
	defaultCentralLimitCPU      = resource.MustParse("1000m")

	defaultScannerAnalyzerRequestMemory = resource.MustParse("100Mi")
	defaultScannerAnalyzerRequestCPU    = resource.MustParse("250m")
	defaultScannerAnalyzerLimitMemory   = resource.MustParse("2500Mi")
	defaultScannerAnalyzerLimitCPU      = resource.MustParse("2000m")

	defaultScannerAnalyzerAutoScaling              = "Enabled"
	defaultScannerAnalyzerScalingReplicas    int32 = 1
	defaultScannerAnalyzerScalingMinReplicas int32 = 1
	defaultScannerAnalyzerScalingMaxReplicas int32 = 3

	defaultScannerDbRequestMemory = resource.MustParse("100Mi")
	defaultScannerDbRequestCPU    = resource.MustParse("250m")
	defaultScannerDbLimitMemory   = resource.MustParse("2500Mi")
	defaultScannerDbLimitCPU      = resource.MustParse("2000m")
)

// PresentManagedDinosaur ...
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
			Auth: private.ManagedCentralAllOfSpecAuth{
				ClientSecret: from.Spec.Auth.ClientSecret,
				ClientId:     from.Spec.Auth.ClientID,
				OwnerUserId:  from.Spec.Auth.OwnerUserID,
				OwnerOrgId:   from.Spec.Auth.OwnerOrgID,
			},
			UiEndpoint: private.ManagedCentralAllOfSpecUiEndpoint{
				Host: from.Spec.Endpoint.Host,
				Tls: private.ManagedCentralAllOfSpecUiEndpointTls{
					Cert: from.Spec.Endpoint.TLS.Cert,
					Key:  from.Spec.Endpoint.TLS.Key,
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
						Cpu:    orDefaultQty(from.Spec.Central.Resources.Requests[corev1.ResourceCPU], defaultCentralRequestCPU).String(),
						Memory: orDefaultQty(from.Spec.Central.Resources.Requests[corev1.ResourceMemory], defaultCentralRequestMemory).String(),
					},
					Limits: private.ResourceList{
						Cpu:    orDefaultQty(from.Spec.Central.Resources.Limits[corev1.ResourceCPU], defaultCentralLimitCPU).String(),
						Memory: orDefaultQty(from.Spec.Central.Resources.Limits[corev1.ResourceMemory], defaultCentralLimitMemory).String(),
					},
				},
			},
			Scanner: private.ManagedCentralAllOfSpecScanner{
				Analyzer: private.ManagedCentralAllOfSpecScannerAnalyzer{
					Scaling: private.ManagedCentralAllOfSpecScannerAnalyzerScaling{
						AutoScaling: orDefaultString(from.Spec.Scanner.Analyzer.Scaling.AutoScaling, defaultScannerAnalyzerAutoScaling),
						Replicas:    orDefaultInt32(from.Spec.Scanner.Analyzer.Scaling.Replicas, defaultScannerAnalyzerScalingReplicas),
						MinReplicas: orDefaultInt32(from.Spec.Scanner.Analyzer.Scaling.MinReplicas, defaultScannerAnalyzerScalingMinReplicas),
						MaxReplicas: orDefaultInt32(from.Spec.Scanner.Analyzer.Scaling.MaxReplicas, defaultScannerAnalyzerScalingMaxReplicas),
					},
					Resources: private.ResourceRequirements{
						Requests: private.ResourceList{
							Cpu:    orDefaultQty(from.Spec.Scanner.Analyzer.Resources.Requests[corev1.ResourceCPU], defaultScannerAnalyzerRequestCPU).String(),
							Memory: orDefaultQty(from.Spec.Scanner.Analyzer.Resources.Requests[corev1.ResourceMemory], defaultScannerAnalyzerRequestMemory).String(),
						},
						Limits: private.ResourceList{
							Cpu:    orDefaultQty(from.Spec.Scanner.Analyzer.Resources.Limits[corev1.ResourceCPU], defaultScannerAnalyzerLimitCPU).String(),
							Memory: orDefaultQty(from.Spec.Scanner.Analyzer.Resources.Limits[corev1.ResourceMemory], defaultScannerAnalyzerLimitMemory).String(),
						},
					},
				},
				Db: private.ManagedCentralAllOfSpecScannerDb{
					// TODO:(create-ticket): add DB configuration values to ManagedCentral Scanner
					Host: "dbhost.rhacs-psql-instance",
					Resources: private.ResourceRequirements{
						Requests: private.ResourceList{
							Cpu:    orDefaultQty(from.Spec.Scanner.Db.Resources.Requests[corev1.ResourceCPU], defaultScannerDbRequestCPU).String(),
							Memory: orDefaultQty(from.Spec.Scanner.Db.Resources.Requests[corev1.ResourceMemory], defaultScannerDbRequestMemory).String(),
						},
						Limits: private.ResourceList{
							Cpu:    orDefaultQty(from.Spec.Scanner.Db.Resources.Limits[corev1.ResourceCPU], defaultScannerDbLimitCPU).String(),
							Memory: orDefaultQty(from.Spec.Scanner.Db.Resources.Limits[corev1.ResourceMemory], defaultScannerDbLimitMemory).String(),
						},
					},
				},
			},
		},
		RequestStatus: from.RequestStatus,
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
