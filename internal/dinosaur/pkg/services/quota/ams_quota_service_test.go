package quota

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/utils"

	"github.com/stretchr/testify/require"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/dinosaurs/types"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"

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
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						cloudAuthorizationResp, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Build()
						return cloudAuthorizationResp, nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						if product != string(ocm.RHACSProduct) {
							return []*v1.QuotaCost{}, nil
						}
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
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
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						if cb.ProductID() == string(ocm.RHACSProduct) {
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
						if product != string(ocm.RHACSProduct) {
							return []*v1.QuotaCost{}, nil
						}
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
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
				ocmClient: &ocm.ClientMock{
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
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return nil, fmt.Errorf("some errors")
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						if product != string(ocm.RHACSProduct) {
							return []*v1.QuotaCost{}, nil
						}
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
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
				Owner: tt.args.owner,
			}
			standard_active, err := quotaService.IsQuotaActive(dinosaur, types.STANDARD)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			eval_active, err := quotaService.IsQuotaActive(dinosaur, types.EVAL)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(standard_active).To(gomega.Equal(tt.args.hasStandardQuota))
			gomega.Expect(eval_active).To(gomega.Equal(tt.args.hasEvalQuota))

			_, err = quotaService.ReserveQuota(emptyCtx, dinosaur, tt.args.dinosaurInstanceType)
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
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return mockClusterAuthorizationResponse(), nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
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
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return mockClusterAuthorizationResponse(), nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb1, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						require.NoError(t, err)
						rrbq2 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
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
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return mockClusterAuthorizationResponse(), nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb1, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						require.NoError(t, err)
						rrbq2 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
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
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return mockClusterAuthorizationResponse(), nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHACSTrialProduct)).ResourceName(resourceName).Cost(0)
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
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return mockClusterAuthorizationResponse(), nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb1, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						require.NoError(t, err)
						rrbq2 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
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
				ocmClient: &ocm.ClientMock{
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
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return mockClusterAuthorizationResponse(), nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string("unknownbillingmodelone")).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb1, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						require.NoError(t, err)
						rrbq2 := v1.NewRelatedResource().BillingModel(string("unknownbillingmodeltwo")).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
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
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						cloudAuthorizationResp, _ := v1.NewClusterAuthorizationResponse().Allowed(false).Build()
						return cloudAuthorizationResp, nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
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
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return mockClusterAuthorizationResponse(), nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHACSTrialProduct)).ResourceName(resourceName).Cost(0)
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
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return mockClusterAuthorizationResponse(), nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHACSTrialProduct)).ResourceName(resourceName).Cost(0)
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
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return mockClusterAuthorizationResponse(), nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHACSTrialProduct)).ResourceName(resourceName).Cost(0)
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
				dinosaurID:     "12231",
				owner:          "testUser",
				cloudAccountID: "cloudAccountID",
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return mockClusterAuthorizationResponse(), nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHACSTrialProduct)).ResourceName(resourceName).Cost(0)
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
			wantBillingModel:              string(v1.BillingModelMarketplace),
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
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return mockClusterAuthorizationResponse(), nil
					},
					GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
						org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
						return org, nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHACSTrialProduct)).ResourceName(resourceName).Cost(0)
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
				CloudAccountID: tt.args.cloudAccountID,
				CloudProvider:  utils.IfThenElse(tt.args.cloudProviderID == "", "cloudProviderID", tt.args.cloudProviderID),
			}
			subID, err := quotaService.ReserveQuota(emptyCtx, dinosaur, types.STANDARD)
			gomega.Expect(subID).To(gomega.Equal(tt.want))
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))

			if tt.wantBillingModel != "" || tt.wantBillingMarketplaceAccount != "" {
				ocmClientMock := tt.fields.ocmClient.(*ocm.ClientMock)
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
				ocmClient: &ocm.ClientMock{
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
				ocmClient: &ocm.ClientMock{
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

func Test_amsQuotaService_IsQuotaActive(t *testing.T) {
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
			ocmClient: &ocm.ClientMock{
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
			ocmClient: &ocm.ClientMock{
				GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
					org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
					return org, nil
				},
				GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
					rrbq1 := v1.NewRelatedResource().BillingModel("unknownbillingmodel").Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
					rrbq2 := v1.NewRelatedResource().BillingModel("unknownbillingmodel2").Product(string(ocm.RHACSTrialProduct)).ResourceName(resourceName).Cost(1)
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
			ocmClient: &ocm.ClientMock{
				GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
					org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
					return org, nil
				},
				GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
					rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
					rrbq2 := v1.NewRelatedResource().BillingModel("unknownbillingmodel2").Product(string(ocm.RHACSTrialProduct)).ResourceName(resourceName).Cost(1)
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
			ocmClient: &ocm.ClientMock{
				GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
					org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
					return org, nil
				},
				GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
					rrbq1 := v1.NewRelatedResource().BillingModel("unknownbillingmodel").Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
					qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
					if err != nil {
						panic("unexpected error")
					}
					rrbq2 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
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
			ocmClient: &ocm.ClientMock{
				GetOrganisationFromExternalIDFunc: func(externalId string) (*v1.Organization, error) {
					org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
					return org, nil
				},
				GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
					rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
					qcb, err := v1.NewQuotaCost().Allowed(0).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
					if err != nil {
						panic("unexpected error")
					}
					rrbq2 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
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
			ocmClient: &ocm.ClientMock{
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
			ocmClient: &ocm.ClientMock{
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
			res, err := quotaService.IsQuotaActive(tt.args.dinosaurRequest, tt.args.dinosaurInstanceType)
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			gomega.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}

func Test_amsQuotaService_CheckQuotaEntitlement(t *testing.T) {
	standardCentral := &dbapi.CentralRequest{
		InstanceType: "standard",
	}

	cloudCentral := &dbapi.CentralRequest{
		InstanceType:   "standard",
		CloudProvider:  awsCloudProvider,
		CloudAccountID: "cloudAccountID",
	}
	const not_allowed = 0
	const not_consumed = 0
	const allowed = 1
	const consumed = 1

	tests := []struct {
		name         string
		amsClient    ocm.AMSClient
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
					makeStandardTestQuotaCost(resourceName, organizationID, allowed, not_consumed, t),
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
					makeStandardTestQuotaCost(resourceName, organizationID, allowed, not_consumed, t),
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
					makeStandardTestQuotaCost(resourceName, organizationID, allowed, not_consumed, t),
					makeStandardTestQuotaCost(resourceName, organizationID, allowed, not_consumed, t),
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
					makeStandardTestQuotaCost(resourceName, organizationID, allowed, not_consumed, t),
					makeStandardTestQuotaCost(resourceName, organizationID, not_allowed, consumed, t),
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
					makeStandardTestQuotaCost(resourceName, organizationID, not_allowed, consumed, t),
					makeStandardTestQuotaCost(resourceName, organizationID, not_allowed, consumed, t),
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
					makeCloudTestQuotaCost(resourceName, organizationID, not_allowed, consumed, t),
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
					makeCloudTestQuotaCost(resourceName, organizationID, allowed, not_consumed, t),
					makeStandardTestQuotaCost(resourceName, organizationID, not_allowed, not_consumed, t),
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
					makeCloudTestQuotaCost(resourceName, organizationID, not_allowed, not_consumed, t),
					makeStandardTestQuotaCost(resourceName, organizationID, not_allowed, not_consumed, t),
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
					makeCloudTestQuotaCost(resourceName, organizationID, allowed, not_consumed, t),
					makeStandardTestQuotaCost(resourceName, organizationID, allowed, not_consumed, t),
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
					makeCloudTestQuotaCost(resourceName, organizationID, allowed, not_consumed, t),
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
				qcb, err := v1.NewQuotaCost().Allowed(allowed).Consumed(not_consumed).OrganizationID(organizationID).RelatedResources(rrbq).Build()
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

			var amsClient ocm.AMSClient = &ocm.ClientMock{
				GetOrganisationFromExternalIDFunc: makeOrganizationFromExtID,
				GetQuotaCostsForProductFunc:       tt.getQuotaFunc,
			}

			quotaServiceFactory := NewDefaultQuotaServiceFactory(amsClient, nil, nil)
			quotaService, _ := quotaServiceFactory.GetQuotaService(api.AMSQuotaType)

			got, err := quotaService.IsQuotaActive(tt.central, types.DinosaurInstanceType(tt.central.InstanceType))
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			if tt.wantErr {
				g.Expect(err.Error()).To(gomega.Equal(tt.wantErrMsg), err.Error())
			}
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func makeStandardTestQuotaCost(resourceName string, organizationID string, allowed int, consumed int, t *testing.T) *v1.QuotaCost {
	rrbq := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
	qcb, err := v1.NewQuotaCost().Allowed(allowed).Consumed(consumed).OrganizationID(organizationID).RelatedResources(rrbq).Build()
	require.NoError(t, err)
	return qcb
}

func makeCloudTestQuotaCost(resourceName string, organizationID string, allowed int, consumed int, t *testing.T) *v1.QuotaCost {
	cloudAccount := v1.NewCloudAccount().CloudAccountID("cloudAccountID").CloudProviderID(awsCloudProvider)
	rrbq := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplaceAWS)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
	qcb, err := v1.NewQuotaCost().Allowed(allowed).Consumed(consumed).OrganizationID(organizationID).RelatedResources(rrbq).CloudAccounts(cloudAccount).Build()
	require.NoError(t, err)
	return qcb
}

func makeOrganizationFromExtID(externalId string) (*v1.Organization, error) {
	org, _ := v1.NewOrganization().ID(fmt.Sprintf("fake-org-id-%s", externalId)).Build()
	return org, nil
}
