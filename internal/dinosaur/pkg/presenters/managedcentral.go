package presenters

import (
	"encoding/json"
	"fmt"
	"sort"
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
			Internal:      from.Internal,
			SecretsStored: getSecretNames(from), // pragma: allowlist secret
		},
		Spec: private.ManagedCentralAllOfSpec{
			Owners: []string{
				from.Owner,
			},
			Auth: private.ManagedCentralAllOfSpecAuth{
				ClientId:     from.AuthConfig.ClientID,
				ClientSecret: from.AuthConfig.ClientSecret, // pragma: allowlist secret
				ClientOrigin: from.AuthConfig.ClientOrigin,
				OwnerOrgId:   from.OrganisationID,
				OwnerOrgName: from.OrganisationName,
				OwnerUserId:  from.OwnerUserID,
				Issuer:       from.AuthConfig.Issuer,
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
			OperatorImage: from.OperatorImage,
			Central: private.ManagedCentralAllOfSpecCentral{
				InstanceType: from.InstanceType,
				Resources: private.ResourceRequirements{
					Requests: map[string]string{
						corev1.ResourceCPU.String():    orDefaultQty(central.Resources.Requests[corev1.ResourceCPU], defaults.Central.CPURequest).String(),
						corev1.ResourceMemory.String(): orDefaultQty(central.Resources.Requests[corev1.ResourceMemory], defaults.Central.MemoryRequest).String(),
					},
					Limits: map[string]string{
						corev1.ResourceCPU.String():    orDefaultQty(central.Resources.Limits[corev1.ResourceCPU], defaults.Central.CPULimit).String(),
						corev1.ResourceMemory.String(): orDefaultQty(central.Resources.Limits[corev1.ResourceMemory], defaults.Central.MemoryLimit).String(),
					},
				},
			},
			Scanner: private.ManagedCentralAllOfSpecScanner{
				Analyzer: private.ManagedCentralAllOfSpecScannerAnalyzer{
					Scaling: private.ManagedCentralAllOfSpecScannerAnalyzerScaling{
						AutoScaling: orDefaultString(scanner.Analyzer.Scaling.AutoScaling, defaults.Scanner.Analyzer.AutoScaling),
						Replicas:    orDefaultInt32(scanner.Analyzer.Scaling.Replicas, defaults.Scanner.Analyzer.Replicas),
						MinReplicas: orDefaultInt32(scanner.Analyzer.Scaling.MinReplicas, defaults.Scanner.Analyzer.MinReplicas),
						MaxReplicas: orDefaultInt32(scanner.Analyzer.Scaling.MaxReplicas, defaults.Scanner.Analyzer.MaxReplicas),
					},
					Resources: private.ResourceRequirements{
						Requests: map[string]string{
							corev1.ResourceCPU.String():    orDefaultQty(scanner.Analyzer.Resources.Requests[corev1.ResourceCPU], defaults.Scanner.Analyzer.CPURequest).String(),
							corev1.ResourceMemory.String(): orDefaultQty(scanner.Analyzer.Resources.Requests[corev1.ResourceMemory], defaults.Scanner.Analyzer.MemoryRequest).String(),
						},
						Limits: map[string]string{
							corev1.ResourceCPU.String():    orDefaultQty(scanner.Analyzer.Resources.Limits[corev1.ResourceCPU], defaults.Scanner.Analyzer.CPULimit).String(),
							corev1.ResourceMemory.String(): orDefaultQty(scanner.Analyzer.Resources.Limits[corev1.ResourceMemory], defaults.Scanner.Analyzer.MemoryLimit).String(),
						},
					},
				},
				Db: private.ManagedCentralAllOfSpecScannerDb{
					Host: "dbhost.rhacs-psql-instance",
					Resources: private.ResourceRequirements{
						Requests: map[string]string{
							corev1.ResourceCPU.String():    orDefaultQty(scanner.Db.Resources.Requests[corev1.ResourceCPU], defaults.Scanner.Db.CPURequest).String(),
							corev1.ResourceMemory.String(): orDefaultQty(scanner.Db.Resources.Requests[corev1.ResourceMemory], defaults.Scanner.Db.MemoryRequest).String(),
						},
						Limits: map[string]string{
							corev1.ResourceCPU.String():    orDefaultQty(scanner.Db.Resources.Limits[corev1.ResourceCPU], defaults.Scanner.Db.CPULimit).String(),
							corev1.ResourceMemory.String(): orDefaultQty(scanner.Db.Resources.Limits[corev1.ResourceMemory], defaults.Scanner.Db.MemoryLimit).String(),
						},
					},
				},
			},
		},
		RequestStatus:  from.Status,
		ForceReconcile: from.ForceReconcile,
	}

	if from.DeletionTimestamp != nil {
		res.Metadata.DeletionTimestamp = from.DeletionTimestamp.Format(time.RFC3339)
	}

	return res
}

// PresentManagedCentralWithSecrets return a private.ManagedCentral including secret data
func (c *ManagedCentralPresenter) PresentManagedCentralWithSecrets(from *dbapi.CentralRequest) private.ManagedCentral {
	managedCentral := c.PresentManagedCentral(from)
	secretInterfaceMap, err := from.Secrets.Object()
	secretStringMap := make(map[string]string, len(secretInterfaceMap))

	if err != nil {
		glog.Errorf("Failed to get Secrets for central request as map %q/%s: %v", from.Name, from.ID, err)
		return managedCentral
	}

	for k, v := range secretInterfaceMap {
		secretStringMap[k] = fmt.Sprintf("%v", v)
	}

	managedCentral.Metadata.Secrets = secretStringMap // pragma: allowlist secret
	return managedCentral
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

func getSecretNames(from *dbapi.CentralRequest) []string {
	secrets, err := from.Secrets.Object()
	if err != nil {
		glog.Errorf("Failed to get Secrets as JSON object for Central request %q/%s: %v", from.Name, from.ClusterID, err)
		return []string{}
	}

	secretNames := make([]string, len(secrets))
	i := 0
	for k := range secrets {
		secretNames[i] = k
		i++
	}

	sort.Strings(secretNames)

	return secretNames
}
