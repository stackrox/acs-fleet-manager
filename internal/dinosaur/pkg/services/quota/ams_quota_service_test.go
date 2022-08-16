package quota

import (
	"fmt"
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/dinosaurs/types"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"

	"github.com/onsi/gomega"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

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
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Build()
						return ca, nil
					},
					GetOrganisationIDFromExternalIDFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						if product != string(ocm.RHACSProduct) {
							return []*v1.QuotaCost{}, nil
						}
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb}, nil
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
							ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Build()
							return ca, nil
						}
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(false).Build()
						return ca, nil
					},
					GetOrganisationIDFromExternalIDFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						if product != string(ocm.RHACSProduct) {
							return []*v1.QuotaCost{}, nil
						}
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb}, nil
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
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(false).Build()
						return ca, nil
					},
					GetOrganisationIDFromExternalIDFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						return []*v1.QuotaCost{}, nil
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
					GetOrganisationIDFromExternalIDFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						if product != string(ocm.RHACSProduct) {
							return []*v1.QuotaCost{}, nil
						}
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb}, nil
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
			sq, err := quotaService.CheckIfQuotaIsDefinedForInstanceType(dinosaur, types.STANDARD)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			eq, err := quotaService.CheckIfQuotaIsDefinedForInstanceType(dinosaur, types.EVAL)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(sq).To(gomega.Equal(tt.args.hasStandardQuota))
			gomega.Expect(eq).To(gomega.Equal(tt.args.hasEvalQuota))

			_, err = quotaService.ReserveQuota(dinosaur, tt.args.dinosaurInstanceType)
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_AMSReserveQuota(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		dinosaurID string
		owner      string
	}
	tests := []struct {
		name             string
		fields           fields
		args             args
		want             string
		wantErr          bool
		wantBillingModel string
	}{
		{
			name: "reserve a quota & get subscription id",
			args: args{
				"12231",
				"testUser",
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						sub := v1.SubscriptionBuilder{}
						sub.ID("1234")
						sub.Status("Active")
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Subscription(&sub).Build()
						return ca, nil
					},
					GetOrganisationIDFromExternalIDFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb}, nil
					},
				},
			},
			wantBillingModel: string(v1.BillingModelMarketplace),
			want:             "1234",
			wantErr:          false,
		},
		{
			name: "when both standard and marketplace billing models are available standard is assigned as billing model",
			args: args{
				"12231",
				"testUser",
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						sub := v1.SubscriptionBuilder{}
						sub.ID("1234")
						sub.Status("Active")
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Subscription(&sub).Build()
						return ca, nil
					},
					GetOrganisationIDFromExternalIDFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb1, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}
						rrbq2 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb2, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq2).Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb1, qcb2}, nil
					},
				},
			},
			wantBillingModel: string(v1.BillingModelStandard),
			want:             "1234",
			wantErr:          false,
		},
		{
			name: "when only marketplace billing model has available resources marketplace billing model is assigned",
			args: args{
				"12231",
				"testUser",
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						sub := v1.SubscriptionBuilder{}
						sub.ID("1234")
						sub.Status("Active")
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Subscription(&sub).Build()
						return ca, nil
					},
					GetOrganisationIDFromExternalIDFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb1, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}
						rrbq2 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb2, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq2).Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb2, qcb1}, nil
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
				"12231",
				"testUser",
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						sub := v1.SubscriptionBuilder{}
						sub.ID("1234")
						sub.Status("Active")
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Subscription(&sub).Build()
						return ca, nil
					},
					GetOrganisationIDFromExternalIDFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHACSTrialProduct)).ResourceName(resourceName).Cost(0)
						qcb1, err := v1.NewQuotaCost().Allowed(0).Consumed(2).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb1}, nil
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
				"12231",
				"testUser",
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						sub := v1.SubscriptionBuilder{}
						sub.ID("1234")
						sub.Status("Active")
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Subscription(&sub).Build()
						return ca, nil
					},
					GetOrganisationIDFromExternalIDFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb1, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}
						rrbq2 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb2, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq2).Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb2, qcb1}, nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "when no quota_costs are available for the given product an error is returned",
			args: args{
				"12231",
				"testUser",
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						sub := v1.SubscriptionBuilder{}
						sub.ID("1234")
						sub.Status("Active")
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Subscription(&sub).Build()
						return ca, nil
					},
					GetOrganisationIDFromExternalIDFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						return []*v1.QuotaCost{}, nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "when the quota_costs returned do not contain a supported billing model an error is returned",
			args: args{
				"12231",
				"testUser",
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						sub := v1.SubscriptionBuilder{}
						sub.ID("1234")
						sub.Status("Active")
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Subscription(&sub).Build()
						return ca, nil
					},
					GetOrganisationIDFromExternalIDFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string("unknownbillingmodelone")).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb1, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}
						rrbq2 := v1.NewRelatedResource().BillingModel(string("unknownbillingmodeltwo")).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb2, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq2).Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb1, qcb2}, nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "failed to reserve a quota",
			args: args{
				"12231",
				"testUser",
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(false).Build()
						return ca, nil
					},
					GetOrganisationIDFromExternalIDFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb}, nil
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
			subID, err := quotaService.ReserveQuota(dinosaur, types.STANDARD)
			gomega.Expect(subID).To(gomega.Equal(tt.want))
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))

			if tt.wantBillingModel != "" {
				ocmClientMock := tt.fields.ocmClient.(*ocm.ClientMock)
				clusterAuthorizationCalls := ocmClientMock.ClusterAuthorizationCalls()
				gomega.Expect(len(clusterAuthorizationCalls)).To(gomega.Equal(1))
				clusterAuthorizationResources := clusterAuthorizationCalls[0].Cb.Resources()
				gomega.Expect(len(clusterAuthorizationResources)).To(gomega.Equal(1))
				clusterAuthorizationResource := clusterAuthorizationResources[0]
				gomega.Expect(string(clusterAuthorizationResource.BillingModel())).To(gomega.Equal(tt.wantBillingModel))
			}
		})
	}
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
						return 0, errors.GeneralError("failed to delete subscription")
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

func Test_amsQuotaService_CheckIfQuotaIsDefinedForInstanceType(t *testing.T) {
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
				GetOrganisationIDFromExternalIDFunc: func(externalId string) (string, error) {
					return fmt.Sprintf("fake-org-id-%s", externalId), nil
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
				GetOrganisationIDFromExternalIDFunc: func(externalId string) (string, error) {
					return fmt.Sprintf("fake-org-id-%s", externalId), nil
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
				GetOrganisationIDFromExternalIDFunc: func(externalId string) (string, error) {
					return fmt.Sprintf("fake-org-id-%s", externalId), nil
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
				GetOrganisationIDFromExternalIDFunc: func(externalId string) (string, error) {
					return fmt.Sprintf("fake-org-id-%s", externalId), nil
				},
				GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
					rrbq1 := v1.NewRelatedResource().BillingModel("unknownbillingmodel").Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
					qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
					if err != nil {
						panic("unexpected error")
					}
					rrbq2 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHACSProduct)).ResourceName(resourceName).Cost(1)
					qcb2, err := v1.NewQuotaCost().Allowed(1).Consumed(2).OrganizationID(organizationID).RelatedResources(rrbq2).Build()
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
			want:    true,
			wantErr: false,
		},
		{
			name: "returns false if there is no supported billing model with an 'allowed' value greater than 0",
			ocmClient: &ocm.ClientMock{
				GetOrganisationIDFromExternalIDFunc: func(externalId string) (string, error) {
					return fmt.Sprintf("fake-org-id-%s", externalId), nil
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
				GetOrganisationIDFromExternalIDFunc: func(externalId string) (string, error) {
					return "", fmt.Errorf("error getting org")
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
				GetOrganisationIDFromExternalIDFunc: func(externalId string) (string, error) {
					return fmt.Sprintf("fake-org-id-%s", externalId), nil
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
			res, err := quotaService.CheckIfQuotaIsDefinedForInstanceType(tt.args.dinosaurRequest, tt.args.dinosaurInstanceType)
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			gomega.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}
