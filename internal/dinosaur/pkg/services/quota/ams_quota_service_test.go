package quota

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/dinosaurs/types"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	ocmImpl "github.com/stackrox/acs-fleet-manager/pkg/client/ocm/impl"
	ocmClientMock "github.com/stackrox/acs-fleet-manager/pkg/client/ocm/mocks"

	"github.com/onsi/gomega"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	serviceErrors "github.com/stackrox/acs-fleet-manager/pkg/errors"

	"github.com/pkg/errors"
)

var emptyCtx = context.Background()

func Test_AMSCheckQuota(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		dinosaurID           string
		reserve              bool
		owner                string
		dinosaurInstanceType types.DinosaurInstanceType
		hasStandardQuota     bool
		hasEvalQuota         bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "owner allowed to reserve quota",
			args: args{
				"",
				false,
				"testUser",
				types.STANDARD,
				true,
				false,
			},
			fields: fields{
				ocmClient: &ocmClientMock.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						cloudAuthorizationResp, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Build()
						return cloudAuthorizationResp, nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						if product != string(ocmImpl.RHACSProduct) {
							return []*v1.QuotaCost{}, nil
						}
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocmImpl.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						require.NoError(t, err)
						return []*v1.QuotaCost{qcb}, nil
					},
					GetCustomerCloudAccountsFunc: func(externalID string, quotaIDs []string) ([]*v1.CloudAccount, error) {
						return []*v1.CloudAccount{}, nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no quota error",
			args: args{
				"",
				false,
				"testUser",
				types.EVAL,
				true,
				false,
			},
			fields: fields{
				ocmClient: &ocmClientMock.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						if cb.ProductID() == string(ocmImpl.RHACSProduct) {
							cloudAuthorizationResp, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Build()
							return cloudAuthorizationResp, nil
						}
						cloudAuthorizationResp, _ := v1.NewClusterAuthorizationResponse().Allowed(false).Build()
						return cloudAuthorizationResp, nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						if product != string(ocmImpl.RHACSProduct) {
							return []*v1.QuotaCost{}, nil
						}
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocmImpl.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						require.NoError(t, err)
						return []*v1.QuotaCost{qcb}, nil
					},
					GetCustomerCloudAccountsFunc: func(externalID string, quotaIDs []string) ([]*v1.CloudAccount, error) {
						return []*v1.CloudAccount{}, nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "owner not allowed to reserve quota",
			args: args{
				"",
				false,
				"testUser",
				types.STANDARD,
				false,
				false,
			},
			fields: fields{
				ocmClient: &ocmClientMock.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						cloudAuthorizationResp, _ := v1.NewClusterAuthorizationResponse().Allowed(false).Build()
						return cloudAuthorizationResp, nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						return []*v1.QuotaCost{}, nil
					},
					GetCustomerCloudAccountsFunc: func(externalID string, quotaIDs []string) ([]*v1.CloudAccount, error) {
						return []*v1.CloudAccount{}, nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "failed to reserve quota",
			args: args{
				"12231",
				false,
				"testUser",
				types.STANDARD,
				true,
				false,
			},
			fields: fields{
				ocmClient: &ocmClientMock.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return nil, fmt.Errorf("some errors")
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						if product != string(ocmImpl.RHACSProduct) {
							return []*v1.QuotaCost{}, nil
						}
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocmImpl.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						require.NoError(t, err)
						return []*v1.QuotaCost{qcb}, nil
					},
					GetCustomerCloudAccountsFunc: func(externalID string, quotaIDs []string) ([]*v1.CloudAccount, error) {
						return []*v1.CloudAccount{}, nil
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			factory := NewDefaultQuotaServiceFactory(tt.fields.ocmClient, nil, nil)
			quotaService, _ := factory.GetQuotaService(api.AMSQuotaType)
			dinosaur := &dbapi.CentralRequest{
				Meta: api.Meta{
					ID: tt.args.dinosaurID,
				},
				Owner:        tt.args.owner,
				InstanceType: string(tt.args.dinosaurInstanceType),
			}
			standardAllowance, err := quotaService.HasQuotaAllowance(dinosaur, types.STANDARD)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			evalAllowance, err := quotaService.HasQuotaAllowance(dinosaur, types.EVAL)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(standardAllowance).To(gomega.Equal(tt.args.hasStandardQuota))
			gomega.Expect(evalAllowance).To(gomega.Equal(tt.args.hasEvalQuota))

			_, err = quotaService.ReserveQuota(emptyCtx, dinosaur, "", "")
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_AMSReserveQuota(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		dinosaurID      string
		owner           string
		cloudAccountID  string
		cloudProviderID string
	}
	tests := []struct {
		name                          string
		fields                        fields
		args                          args
		want                          string
		wantErr                       bool
		wantBillingModel              string
		wantBillingMarketplaceAccount string
	}{
		{
			name: "reserve a quota & get subscription id",
			args: args{
				dinosaurID: "12231",
				owner:      "testUser",
			},
			fields: fields{
				ocmClient: &ocmClientMock.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return mockClusterAuthorizationResponse(), nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocmImpl.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						require.NoError(t, err)
						return []*v1.QuotaCost{qcb}, nil
					},
					GetCustomerCloudAccountsFunc: func(externalID string, quotaIDs []string) ([]*v1.CloudAccount, error) {
						return []*v1.CloudAccount{}, nil
					},
				},
			},
			wantBillingModel: string(v1.BillingModelMarketplace),
			want:             "1234",
			wantErr:          false,
		},
		{
			name: "when both standard and marketplace billing models are available marketplace is assigned as billing model",
			args: args{
				dinosaurID: "12231",
				owner:      "testUser",
			},
			fields: fields{
				ocmClient: &ocmClientMock.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return mockClusterAuthorizationResponse(), nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocmImpl.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb1, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						require.NoError(t, err)
						rrbq2 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocmImpl.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb2, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq2).Build()
						require.NoError(t, err)
						return []*v1.QuotaCost{qcb1, qcb2}, nil
					},
					GetCustomerCloudAccountsFunc: func(externalID string, quotaIDs []string) ([]*v1.CloudAccount, error) {
						return []*v1.CloudAccount{}, nil
					},
				},
			},
			wantBillingModel: string(v1.BillingModelMarketplace),
			want:             "1234",
			wantErr:          false,
		},
		{
			name: "when only marketplace billing model has available resources marketplace billing model is assigned",
			args: args{
				dinosaurID: "12231",
				owner:      "testUser",
			},
			fields: fields{
				ocmClient: &ocmClientMock.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return mockClusterAuthorizationResponse(), nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocmImpl.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb1, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						require.NoError(t, err)
						rrbq2 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocmImpl.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb2, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq2).Build()
						require.NoError(t, err)
						return []*v1.QuotaCost{qcb2, qcb1}, nil
					},
					GetCustomerCloudAccountsFunc: func(externalID string, quotaIDs []string) ([]*v1.CloudAccount, error) {
						return []*v1.CloudAccount{}, nil
					},
				},
			},
			wantBillingModel: string(v1.BillingModelMarketplace),
			want:             "1234",
			wantErr:          false,
		},
		{
			name: "when a related resource has a supported billing model with cost of 0 that billing model is allowed",
			args: args{
				dinosaurID: "12231",
				owner:      "testUser",
			},
			fields: fields{
				ocmClient: &ocmClientMock.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return mockClusterAuthorizationResponse(), nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocmImpl.RHACSTrialProduct)).ResourceName(resourceName).Cost(0)
						qcb1, err := v1.NewQuotaCost().Allowed(0).Consumed(2).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						require.NoError(t, err)
						return []*v1.QuotaCost{qcb1}, nil
					},
					GetCustomerCloudAccountsFunc: func(externalID string, quotaIDs []string) ([]*v1.CloudAccount, error) {
						return []*v1.CloudAccount{}, nil
					},
				},
			},
			wantBillingModel: string(v1.BillingModelMarketplace),
			want:             "1234",
			wantErr:          false,
		},
		{
			name: "when all matching quota_costs consumed resources are higher or equal than the allowed resources an error is returned",
			args: args{
				dinosaurID: "12231",
				owner:      "testUser",
			},
			fields: fields{
				ocmClient: &ocmClientMock.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return mockClusterAuthorizationResponse(), nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocmImpl.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb1, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						require.NoError(t, err)
						rrbq2 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocmImpl.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb2, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq2).Build()
						require.NoError(t, err)
						return []*v1.QuotaCost{qcb2, qcb1}, nil
					},
					GetCustomerCloudAccountsFunc: func(externalID string, quotaIDs []string) ([]*v1.CloudAccount, error) {
						return []*v1.CloudAccount{}, nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "when no quota_costs are available for the given product an error is returned",
			args: args{
				dinosaurID: "12231",
				owner:      "testUser",
			},
			fields: fields{
				ocmClient: &ocmClientMock.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return mockClusterAuthorizationResponse(), nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						return []*v1.QuotaCost{}, nil
					},
					GetCustomerCloudAccountsFunc: func(externalID string, quotaIDs []string) ([]*v1.CloudAccount, error) {
						return []*v1.CloudAccount{}, nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "when the quota_costs returned do not contain a supported billing model an error is returned",
			args: args{
				dinosaurID: "12231",
				owner:      "testUser",
			},
			fields: fields{
				ocmClient: &ocmClientMock.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return mockClusterAuthorizationResponse(), nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string("unknownbillingmodelone")).Product(string(ocmImpl.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb1, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						require.NoError(t, err)
						rrbq2 := v1.NewRelatedResource().BillingModel(string("unknownbillingmodeltwo")).Product(string(ocmImpl.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb2, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq2).Build()
						require.NoError(t, err)
						return []*v1.QuotaCost{qcb1, qcb2}, nil
					},
					GetCustomerCloudAccountsFunc: func(externalID string, quotaIDs []string) ([]*v1.CloudAccount, error) {
						return []*v1.CloudAccount{}, nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "failed to reserve a quota",
			args: args{
				dinosaurID: "12231",
				owner:      "testUser",
			},
			fields: fields{
				ocmClient: &ocmClientMock.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						cloudAuthorizationResp, _ := v1.NewClusterAuthorizationResponse().Allowed(false).Build()
						return cloudAuthorizationResp, nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocmImpl.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						require.NoError(t, err)
						return []*v1.QuotaCost{qcb}, nil
					},
					GetCustomerCloudAccountsFunc: func(externalID string, quotaIDs []string) ([]*v1.CloudAccount, error) {
						return []*v1.CloudAccount{}, nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "failed to get cloud accounts",
			args: args{
				dinosaurID: "12231",
				owner:      "testUser",
			},
			fields: fields{
				ocmClient: &ocmClientMock.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return mockClusterAuthorizationResponse(), nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocmImpl.RHACSTrialProduct)).ResourceName(resourceName).Cost(0)
						qcb1, err := v1.NewQuotaCost().Allowed(0).Consumed(2).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						require.NoError(t, err)
						return []*v1.QuotaCost{qcb1}, nil
					},
					GetCustomerCloudAccountsFunc: func(externalID string, quotaIDs []string) ([]*v1.CloudAccount, error) {
						return nil, errors.New("unsuccessful cloud accounts test call")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "cloud account id in request is empty while cloud_accounts response is not results in error",
			args: args{
				dinosaurID: "12231",
				owner:      "testUser",
			},
			fields: fields{
				ocmClient: &ocmClientMock.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return mockClusterAuthorizationResponse(), nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocmImpl.RHACSTrialProduct)).ResourceName(resourceName).Cost(0)
						qcb1, err := v1.NewQuotaCost().Allowed(0).Consumed(2).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						require.NoError(t, err)
						return []*v1.QuotaCost{qcb1}, nil
					},
					GetCustomerCloudAccountsFunc: func(externalID string, quotaIDs []string) ([]*v1.CloudAccount, error) {
						cloudAccount, _ := v1.NewCloudAccount().
							CloudAccountID("cloudAccountID").
							CloudProviderID("cloudProviderID").
							Build()
						return []*v1.CloudAccount{
							cloudAccount,
						}, nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "cloud account id in request does not match ids in cloud_accounts response results in error",
			args: args{
				dinosaurID:     "12231",
				owner:          "testUser",
				cloudAccountID: "different cloudAccountID",
			},
			fields: fields{
				ocmClient: &ocmClientMock.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return mockClusterAuthorizationResponse(), nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocmImpl.RHACSTrialProduct)).ResourceName(resourceName).Cost(0)
						qcb1, err := v1.NewQuotaCost().Allowed(0).Consumed(2).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						require.NoError(t, err)
						return []*v1.QuotaCost{qcb1}, nil
					},
					GetCustomerCloudAccountsFunc: func(externalID string, quotaIDs []string) ([]*v1.CloudAccount, error) {
						cloudAccount, _ := v1.NewCloudAccount().
							CloudAccountID("cloudAccountID").
							CloudProviderID("cloudProviderID").
							Build()
						return []*v1.CloudAccount{
							cloudAccount,
						}, nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "cloud account matches cloud_accounts response results in successful call",
			args: args{
				dinosaurID:      "12231",
				owner:           "testUser",
				cloudAccountID:  "cloudAccountID",
				cloudProviderID: "aws",
			},
			fields: fields{
				ocmClient: &ocmClientMock.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return mockClusterAuthorizationResponse(), nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().CloudProvider("aws").BillingModel(string(v1.BillingModelMarketplaceAWS)).Product(string(ocmImpl.RHACSTrialProduct)).ResourceName(resourceName).Cost(0)
						qcb1, err := v1.NewQuotaCost().Allowed(0).Consumed(2).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						require.NoError(t, err)
						return []*v1.QuotaCost{qcb1}, nil
					},
					GetCustomerCloudAccountsFunc: func(externalID string, quotaIDs []string) ([]*v1.CloudAccount, error) {
						cloudAccount, _ := v1.NewCloudAccount().
							CloudAccountID("cloudAccountID").
							CloudProviderID("aws").
							Build()
						return []*v1.CloudAccount{
							cloudAccount,
						}, nil
					},
				},
			},
			wantBillingModel:              string(v1.BillingModelMarketplaceAWS),
			wantBillingMarketplaceAccount: "cloudAccountID",
			want:                          "1234",
			wantErr:                       false,
		},
		{
			name: "aws cloud provider results in marketplace-aws billing model",
			args: args{
				dinosaurID:      "12231",
				owner:           "testUser",
				cloudAccountID:  "cloudAccountID",
				cloudProviderID: "aws",
			},
			fields: fields{
				ocmClient: &ocmClientMock.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return mockClusterAuthorizationResponse(), nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocmImpl.RHACSTrialProduct)).ResourceName(resourceName).Cost(0)
						qcb1, err := v1.NewQuotaCost().Allowed(0).Consumed(2).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						require.NoError(t, err)
						return []*v1.QuotaCost{qcb1}, nil
					},
					GetCustomerCloudAccountsFunc: func(externalID string, quotaIDs []string) ([]*v1.CloudAccount, error) {
						cloudAccount, _ := v1.NewCloudAccount().
							CloudAccountID("cloudAccountID").
							CloudProviderID("aws").
							Build()
						return []*v1.CloudAccount{
							cloudAccount,
						}, nil
					},
				},
			},
			wantBillingModel:              string(v1.BillingModelMarketplaceAWS),
			wantBillingMarketplaceAccount: "cloudAccountID",
			want:                          "1234",
			wantErr:                       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			factory := NewDefaultQuotaServiceFactory(tt.fields.ocmClient, nil, nil)
			quotaService, _ := factory.GetQuotaService(api.AMSQuotaType)
			dinosaur := &dbapi.CentralRequest{
				Meta: api.Meta{
					ID: tt.args.dinosaurID,
				},
				Owner:          tt.args.owner,
				InstanceType:   string(types.STANDARD),
				CloudAccountID: tt.args.cloudAccountID,
				CloudProvider:  utils.IfThenElse(tt.args.cloudProviderID == "", "cloudProviderID", tt.args.cloudProviderID),
			}
			subID, err := quotaService.ReserveQuota(emptyCtx, dinosaur, "", "")
			gomega.Expect(subID).To(gomega.Equal(tt.want))
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))

			if tt.wantBillingModel != "" || tt.wantBillingMarketplaceAccount != "" {
				ocmClientMock := tt.fields.ocmClient.(*ocmClientMock.ClientMock)
				clusterAuthorizationCalls := ocmClientMock.ClusterAuthorizationCalls()
				gomega.Expect(len(clusterAuthorizationCalls)).To(gomega.Equal(1))
				clusterAuthorizationResources := clusterAuthorizationCalls[0].Cb.Resources()
				gomega.Expect(len(clusterAuthorizationResources)).To(gomega.Equal(1))
				clusterAuthorizationResource := clusterAuthorizationResources[0]
				if tt.wantBillingModel != "" {
					gomega.Expect(string(clusterAuthorizationResource.BillingModel())).To(gomega.Equal(tt.wantBillingModel))
				}
				if tt.wantBillingMarketplaceAccount != "" {
					gomega.Expect(clusterAuthorizationResource.BillingMarketplaceAccount()).To(gomega.Equal(tt.wantBillingMarketplaceAccount))
				}
			}
		})
	}
}

func mockClusterAuthorizationResponse() *v1.ClusterAuthorizationResponse {
	sub := v1.SubscriptionBuilder{}
	sub.ID("1234")
	sub.Status("Active")
	cloudAuthorizationResp, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Subscription(&sub).Build()
	return cloudAuthorizationResp
}

func Test_Delete_Quota(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		subscriptionID string
	}
	tests := []struct {
		// name is just a description of the test
		name   string
		fields fields
		args   args
		// want (there can be more than one) is the outputs that we expect, they can be compared after the test
		// function has been executed
		// wantErr is similar to want, but instead of testing the actual returned error, we're just testing than any
		// error has been returned
		wantErr bool
	}{
		{
			name: "delete a quota by id",
			args: args{
				subscriptionID: "1223",
			},
			fields: fields{
				ocmClient: &ocmClientMock.ClientMock{
					DeleteSubscriptionFunc: func(id string) (int, error) {
						return 1, nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "failed to delete a quota by id",
			args: args{
				subscriptionID: "1223",
			},
			fields: fields{
				ocmClient: &ocmClientMock.ClientMock{
					DeleteSubscriptionFunc: func(id string) (int, error) {
						return 0, serviceErrors.GeneralError("failed to delete subscription")
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewDefaultQuotaServiceFactory(tt.fields.ocmClient, nil, nil)
			quotaService, _ := factory.GetQuotaService(api.AMSQuotaType)
			err := quotaService.DeleteQuota(tt.args.subscriptionID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteQuota() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_amsQuotaService_HasQuotaAllowance(t *testing.T) {
	type args struct {
		dinosaurRequest      *dbapi.CentralRequest
		dinosaurInstanceType types.DinosaurInstanceType
	}

	tests := []struct {
		name      string
		ocmClient ocm.Client
		args      args
		want      bool
		wantErr   bool
	}{
		{
			name: "returns false if no quota cost exists for the dinosaur's organization",
			ocmClient: &ocmClientMock.ClientMock{
				GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
					org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
					return org, nil
				},
				GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
					return []*v1.QuotaCost{}, nil
				},
			},
			args: args{
				dinosaurRequest:      &dbapi.CentralRequest{OrganisationID: "dinosaur-org-1"},
				dinosaurInstanceType: types.STANDARD,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "returns false if the quota cost billing model is not among the supported ones",
			ocmClient: &ocmClientMock.ClientMock{
				GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
					org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
					return org, nil
				},
				GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
					rrbq1 := v1.NewRelatedResource().BillingModel("unknownbillingmodel").Product(string(ocmImpl.RHACSProduct)).ResourceName(resourceName).Cost(1)
					rrbq2 := v1.NewRelatedResource().BillingModel("unknownbillingmodel2").Product(string(ocmImpl.RHACSTrialProduct)).ResourceName(resourceName).Cost(1)
					qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1, rrbq2).Build()
					if err != nil {
						panic("unexpected error")
					}
					return []*v1.QuotaCost{qcb}, nil
				},
			},
			args: args{
				dinosaurRequest:      &dbapi.CentralRequest{OrganisationID: "dinosaur-org-1"},
				dinosaurInstanceType: types.STANDARD,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "returns true if there is at least a 'standard' quota cost billing model",
			ocmClient: &ocmClientMock.ClientMock{
				GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
					org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
					return org, nil
				},
				GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
					rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocmImpl.RHACSProduct)).ResourceName(resourceName).Cost(1)
					rrbq2 := v1.NewRelatedResource().BillingModel("unknownbillingmodel2").Product(string(ocmImpl.RHACSTrialProduct)).ResourceName(resourceName).Cost(1)
					qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1, rrbq2).Build()
					if err != nil {
						panic("unexpected error")
					}
					return []*v1.QuotaCost{qcb}, nil
				},
			},
			args: args{
				dinosaurRequest:      &dbapi.CentralRequest{OrganisationID: "dinosaur-org-1"},
				dinosaurInstanceType: types.STANDARD,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "returns true if there is at least a 'marketplace' quota cost billing model",
			ocmClient: &ocmClientMock.ClientMock{
				GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
					org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
					return org, nil
				},
				GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
					rrbq1 := v1.NewRelatedResource().BillingModel("unknownbillingmodel").Product(string(ocmImpl.RHACSProduct)).ResourceName(resourceName).Cost(1)
					qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
					if err != nil {
						panic("unexpected error")
					}
					rrbq2 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocmImpl.RHACSProduct)).ResourceName(resourceName).Cost(1)
					cloudAccount := v1.NewCloudAccount().CloudAccountID("cloudAccountID").CloudProviderID(awsCloudProvider)
					qcb2, err := v1.NewQuotaCost().Allowed(1).Consumed(2).OrganizationID(organizationID).RelatedResources(rrbq2).CloudAccounts(cloudAccount).Build()
					if err != nil {
						panic("unexpected error")
					}

					return []*v1.QuotaCost{qcb, qcb2}, nil
				},
			},
			args: args{
				dinosaurRequest: &dbapi.CentralRequest{OrganisationID: "dinosaur-org-1",
					CloudProvider:  awsCloudProvider,
					CloudAccountID: "cloudAccountID",
				},
				dinosaurInstanceType: types.STANDARD,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "returns false if there is no supported billing model with an 'allowed' value greater than 0",
			ocmClient: &ocmClientMock.ClientMock{
				GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
					org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
					return org, nil
				},
				GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
					rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocmImpl.RHACSProduct)).ResourceName(resourceName).Cost(1)
					qcb, err := v1.NewQuotaCost().Allowed(0).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
					if err != nil {
						panic("unexpected error")
					}
					rrbq2 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocmImpl.RHACSProduct)).ResourceName(resourceName).Cost(1)
					qcb2, err := v1.NewQuotaCost().Allowed(0).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq2).Build()
					if err != nil {
						panic("unexpected error")
					}
					return []*v1.QuotaCost{qcb, qcb2}, nil
				},
			},
			args: args{
				dinosaurRequest:      &dbapi.CentralRequest{OrganisationID: "dinosaur-org-1"},
				dinosaurInstanceType: types.STANDARD,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "returns an error if it fails retrieving the organization ID",
			ocmClient: &ocmClientMock.ClientMock{
				GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
					return nil, fmt.Errorf("error getting org")
				},
				GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
					return []*v1.QuotaCost{}, nil
				},
			},
			args: args{
				dinosaurRequest:      &dbapi.CentralRequest{OrganisationID: "dinosaur-org-1"},
				dinosaurInstanceType: types.STANDARD,
			},
			wantErr: true,
		},
		{
			name: "returns an error if it fails retrieving quota costs",
			ocmClient: &ocmClientMock.ClientMock{
				GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
					org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
					return org, nil
				},
				GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
					return []*v1.QuotaCost{}, fmt.Errorf("error getting quota costs")
				},
			},
			args: args{
				dinosaurRequest:      &dbapi.CentralRequest{OrganisationID: "dinosaur-org-1"},
				dinosaurInstanceType: types.STANDARD,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			quotaServiceFactory := NewDefaultQuotaServiceFactory(tt.ocmClient, nil, nil)
			quotaService, _ := quotaServiceFactory.GetQuotaService(api.AMSQuotaType)
			res, err := quotaService.HasQuotaAllowance(tt.args.dinosaurRequest, tt.args.dinosaurInstanceType)
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			gomega.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}

func Test_amsQuotaService_HasQuotaAllowance_Extra(t *testing.T) {
	standardCentral := &dbapi.CentralRequest{
		InstanceType: "standard",
	}

	cloudCentral := &dbapi.CentralRequest{
		InstanceType:   "standard",
		CloudProvider:  awsCloudProvider,
		CloudAccountID: "cloudAccountID",
	}
	const notAllowed = 0
	const notConsumed = 0
	const allowed = 1
	const consumed = 1

	tests := []struct {
		name         string
		amsClient    ocmImpl.AMSClient
		getQuotaFunc func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error)
		central      *dbapi.CentralRequest

		want       bool
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "returns true for single allowed quota cost",
			getQuotaFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
				return []*v1.QuotaCost{
					makeStandardTestQuotaCost(resourceName, organizationID, allowed, notConsumed, t),
				}, nil
			},
			central: standardCentral,
			want:    true,
			wantErr: false,
		},
		{
			name: "returns true for single allowed quota cost for eval instance",
			getQuotaFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
				return []*v1.QuotaCost{
					makeStandardTestQuotaCost(resourceName, organizationID, allowed, notConsumed, t),
				}, nil
			},
			central: &dbapi.CentralRequest{
				InstanceType: "eval",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "returns false for no quota cost",
			getQuotaFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
				return []*v1.QuotaCost{}, nil
			},
			central: standardCentral,
			want:    false,
			wantErr: false,
		},
		{
			name: "returns true for several allowed quota costs",
			getQuotaFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
				return []*v1.QuotaCost{
					makeStandardTestQuotaCost(resourceName, organizationID, allowed, notConsumed, t),
					makeStandardTestQuotaCost(resourceName, organizationID, allowed, notConsumed, t),
				}, nil
			},
			central: standardCentral,
			want:    true,
			wantErr: false,
		},
		{
			name: "returns true for one of several allowed quota costs",
			getQuotaFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
				return []*v1.QuotaCost{
					makeStandardTestQuotaCost(resourceName, organizationID, allowed, notConsumed, t),
					makeStandardTestQuotaCost(resourceName, organizationID, notAllowed, consumed, t),
				}, nil
			},
			central: standardCentral,
			want:    true,
			wantErr: false,
		},
		{
			name: "returns true if organisation has exceeded their quota limits but entitlement is still active",
			getQuotaFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
				return []*v1.QuotaCost{
					makeStandardTestQuotaCost(resourceName, organizationID, allowed, consumed*2, t),
					makeStandardTestQuotaCost(resourceName, organizationID, allowed, consumed*3, t),
				}, nil
			},
			central: standardCentral,
			want:    true,
			wantErr: false,
		},
		{
			name: "returns false for no quota cost allowed",
			getQuotaFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
				return []*v1.QuotaCost{
					makeStandardTestQuotaCost(resourceName, organizationID, notAllowed, consumed, t),
					makeStandardTestQuotaCost(resourceName, organizationID, notAllowed, consumed, t),
				}, nil
			},
			central: standardCentral,
			want:    false,
			wantErr: false,
		},
		{
			name: "returns false if cloud account has no allowed cost, but standard has",
			getQuotaFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
				return []*v1.QuotaCost{
					makeCloudTestQuotaCost(resourceName, organizationID, notAllowed, consumed, t),
					makeStandardTestQuotaCost(resourceName, organizationID, allowed, consumed, t),
				}, nil
			},
			central: cloudCentral,
			want:    false,
			wantErr: false,
		},
		{
			name: "returns false if standard account has no allowed cost, but cloud has",
			getQuotaFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
				return []*v1.QuotaCost{
					makeCloudTestQuotaCost(resourceName, organizationID, allowed, notConsumed, t),
					makeStandardTestQuotaCost(resourceName, organizationID, notAllowed, notConsumed, t),
				}, nil
			},
			central: standardCentral,
			want:    false,
			wantErr: false,
		},
		{
			name: "returns false if cloud account has no allowed cost, neither standard has",
			getQuotaFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
				return []*v1.QuotaCost{
					makeCloudTestQuotaCost(resourceName, organizationID, notAllowed, notConsumed, t),
					makeStandardTestQuotaCost(resourceName, organizationID, notAllowed, notConsumed, t),
				}, nil
			},
			central: cloudCentral,
			want:    false,
			wantErr: false,
		},
		{
			name: "returns true if cloud account has allowed cost, and standard has",
			getQuotaFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
				return []*v1.QuotaCost{
					makeCloudTestQuotaCost(resourceName, organizationID, allowed, notConsumed, t),
					makeStandardTestQuotaCost(resourceName, organizationID, allowed, notConsumed, t),
				}, nil
			},
			central: cloudCentral,
			want:    true,
			wantErr: false,
		},
		{
			name: "returns false if cloud account has no active account",
			getQuotaFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
				return []*v1.QuotaCost{
					makeCloudTestQuotaCost(resourceName, organizationID, allowed, notConsumed, t),
				}, nil
			},
			central: &dbapi.CentralRequest{
				InstanceType:   "standard",
				CloudProvider:  awsCloudProvider,
				CloudAccountID: "unsubscribedCloudAccountID",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "returns an error when it fails to get quota costs from ams",
			getQuotaFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
				return nil, fmt.Errorf("failed to get quota cost")
			},
			central:    standardCentral,
			want:       false,
			wantErr:    true,
			wantErrMsg: "RHACS-MGMT-9: failed to get assigned quota of type \"standard\" for organization with external id \"\" and id \"fake-org-id-\"\n caused by: failed to get quota cost",
		},
		{
			name: "returns an error when it finds only unsupported billing models",
			getQuotaFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
				rrbq := v1.NewRelatedResource().BillingModel("unsupported").Product(product).ResourceName(resourceName).Cost(1)
				qcb, err := v1.NewQuotaCost().Allowed(allowed).Consumed(notConsumed).OrganizationID(organizationID).RelatedResources(rrbq).Build()
				require.NoError(t, err)
				return []*v1.QuotaCost{qcb}, nil
			},
			central:    standardCentral,
			want:       false,
			wantErr:    true,
			wantErrMsg: "RHACS-MGMT-9: found only unsupported billing models [\"unsupported\"] for product \"RHACS\"",
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			var amsClient ocmImpl.AMSClient = &ocmClientMock.ClientMock{
				GetOrganisationFromExternalIDFunc: makeOrganizationFromExternalID,
				GetQuotaCostsForProductFunc:       tt.getQuotaFunc,
			}

			quotaServiceFactory := NewDefaultQuotaServiceFactory(amsClient, nil, nil)
			quotaService, _ := quotaServiceFactory.GetQuotaService(api.AMSQuotaType)

			got, err := quotaService.HasQuotaAllowance(tt.central, types.DinosaurInstanceType(tt.central.InstanceType))
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			if tt.wantErr {
				g.Expect(err.Error()).To(gomega.Equal(tt.wantErrMsg), err.Error())
			}
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func makeStandardTestQuotaCost(resourceName string, organizationID string, allowed int, consumed int, t *testing.T) *v1.QuotaCost {
	rrbq := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocmImpl.RHACSProduct)).ResourceName(resourceName).Cost(1)
	qcb, err := v1.NewQuotaCost().Allowed(allowed).Consumed(consumed).OrganizationID(organizationID).RelatedResources(rrbq).Build()
	require.NoError(t, err)
	return qcb
}

func makeCloudTestQuotaCost(resourceName string, organizationID string, allowed int, consumed int, t *testing.T) *v1.QuotaCost {
	cloudAccount := v1.NewCloudAccount().CloudAccountID("cloudAccountID").CloudProviderID(awsCloudProvider)
	rrbq := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplaceAWS)).Product(string(ocmImpl.RHACSProduct)).ResourceName(resourceName).Cost(1)
	qcb, err := v1.NewQuotaCost().Allowed(allowed).Consumed(consumed).OrganizationID(organizationID).RelatedResources(rrbq).CloudAccounts(cloudAccount).Build()
	require.NoError(t, err)
	return qcb
}

func makeOrganizationFromExternalID(externalId string) (*v1.Organization, error) {
	org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
	return org, nil
}

func TestMapAllowedQuotaCosts(t *testing.T) {
	var nilStringArray []string

	type testCase struct {
		quotaCostBuilders   []*v1.QuotaCostBuilder
		expectedAllowed     map[v1.BillingModel]int
		expectedUnsupported []string
	}

	for name, testcase := range map[string]testCase{
		"empty": {
			quotaCostBuilders:   []*v1.QuotaCostBuilder{},
			expectedAllowed:     map[v1.BillingModel]int{},
			expectedUnsupported: nilStringArray,
		},
		"one allowed": {
			quotaCostBuilders: []*v1.QuotaCostBuilder{v1.NewQuotaCost().Allowed(1).RelatedResources(
				v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)),
			)},
			expectedAllowed:     map[v1.BillingModel]int{v1.BillingModelStandard: 1},
			expectedUnsupported: nilStringArray,
		},
		"one allowed and one unsupported": {
			quotaCostBuilders: []*v1.QuotaCostBuilder{
				v1.NewQuotaCost().Allowed(1).RelatedResources(
					v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)),
					v1.NewRelatedResource().BillingModel("unsupported"),
				),
			},
			expectedAllowed:     map[v1.BillingModel]int{v1.BillingModelStandard: 1},
			expectedUnsupported: []string{"unsupported"},
		},
		"one zero allowed and one 10 allowed": {
			quotaCostBuilders: []*v1.QuotaCostBuilder{
				v1.NewQuotaCost().Allowed(0).RelatedResources(
					v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)),
				),
				v1.NewQuotaCost().Allowed(10).RelatedResources(
					v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplaceAWS)),
				),
			},
			expectedAllowed:     map[v1.BillingModel]int{v1.BillingModelMarketplaceAWS: 10},
			expectedUnsupported: nilStringArray,
		},
		"zero allowed and one unsupported": {
			quotaCostBuilders: []*v1.QuotaCostBuilder{
				v1.NewQuotaCost().Allowed(0).RelatedResources(
					v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)),
				),
				v1.NewQuotaCost().Allowed(1).RelatedResources(
					v1.NewRelatedResource().BillingModel("unsupported"),
				),
			}, expectedAllowed: map[v1.BillingModel]int{},
			expectedUnsupported: []string{"unsupported"},
		},
		"no allowed and one unsupported": {
			quotaCostBuilders: []*v1.QuotaCostBuilder{
				v1.NewQuotaCost().Allowed(1).RelatedResources(
					v1.NewRelatedResource().BillingModel("unsupported"),
				),
			},
			expectedAllowed:     map[v1.BillingModel]int{},
			expectedUnsupported: []string{"unsupported"},
		},
		"many allowed and many unsupported": {
			quotaCostBuilders: []*v1.QuotaCostBuilder{
				v1.NewQuotaCost().Allowed(10).RelatedResources(
					v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)),
					v1.NewRelatedResource().BillingModel("unsupported1"),
				),
				v1.NewQuotaCost().Allowed(1).RelatedResources(
					v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)),
					v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplaceAWS)),
					v1.NewRelatedResource().BillingModel("unsupported2"),
					v1.NewRelatedResource().BillingModel("unsupported3"),
				),
			},
			expectedAllowed: map[v1.BillingModel]int{
				v1.BillingModelStandard:       11,
				v1.BillingModelMarketplaceAWS: 1,
			},
			expectedUnsupported: []string{"unsupported1", "unsupported2", "unsupported3"},
		},
	} {
		t.Run(name, func(tt *testing.T) {
			var quotaCosts []*v1.QuotaCost = make([]*v1.QuotaCost, len(testcase.quotaCostBuilders))
			for _, qc := range testcase.quotaCostBuilders {
				q, _ := qc.Build()
				quotaCosts = append(quotaCosts, q)
			}
			allowed, unsupported := mapAllowedQuotaCosts(quotaCosts)
			assert.Equal(tt, len(testcase.expectedAllowed), len(allowed))
			for model, costs := range allowed {
				var n int
				for _, cost := range costs {
					n += cost.Allowed()
				}
				assert.Equal(tt, testcase.expectedAllowed[model], n)
			}
			assert.Equal(tt, testcase.expectedUnsupported, unsupported)
		})
	}
}

func TestCloudAccountIsActive(t *testing.T) {

	type testCase struct {
		quotaCostBuilders []*v1.QuotaCostBuilder
		central           *dbapi.CentralRequest
		expectedActive    bool
	}

	testAWSQuotaCostBuilder := v1.NewQuotaCost().Allowed(1).CloudAccounts(
		v1.NewCloudAccount().CloudAccountID("test-id").CloudProviderID("test-provider"),
	).RelatedResources(v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplaceAWS)))

	for name, testcase := range map[string]testCase{
		"no cloud account, no cost": {
			central:        &dbapi.CentralRequest{},
			expectedActive: false,
		},
		"no cloud account, yes cost": {
			quotaCostBuilders: []*v1.QuotaCostBuilder{testAWSQuotaCostBuilder},
			central:           &dbapi.CentralRequest{},
			expectedActive:    false,
		},
		"yes cloud account, yes cost": {
			quotaCostBuilders: []*v1.QuotaCostBuilder{testAWSQuotaCostBuilder},
			central: &dbapi.CentralRequest{
				CloudAccountID: "test-id",
				CloudProvider:  "test-provider",
			},
			expectedActive: true,
		},
		"wrong cloud account": {
			quotaCostBuilders: []*v1.QuotaCostBuilder{testAWSQuotaCostBuilder},
			central: &dbapi.CentralRequest{
				CloudAccountID: "test-id-1",
				CloudProvider:  "test-provider",
			},
			expectedActive: false,
		},
		"wrong cloud provider": {
			quotaCostBuilders: []*v1.QuotaCostBuilder{testAWSQuotaCostBuilder},
			central: &dbapi.CentralRequest{
				CloudAccountID: "test-id",
				CloudProvider:  "test-provider-1",
			},
			expectedActive: false,
		},
	} {
		t.Run(name, func(tt *testing.T) {
			var quotaCosts []*v1.QuotaCost = make([]*v1.QuotaCost, len(testcase.quotaCostBuilders))
			for _, qc := range testcase.quotaCostBuilders {
				q, _ := qc.Build()
				quotaCosts = append(quotaCosts, q)
			}
			allowed, _ := mapAllowedQuotaCosts(quotaCosts)
			active := cloudAccountIsActive(allowed, testcase.central)
			assert.Equal(tt, testcase.expectedActive, active)
		})
	}
}
