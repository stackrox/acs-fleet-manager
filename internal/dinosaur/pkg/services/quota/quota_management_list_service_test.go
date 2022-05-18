package quota

import (
	"github.com/stackrox/acs-fleet-manager/pkg/quota_management"
	"net/http"
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/dinosaurs/types"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/onsi/gomega"
	mocket "github.com/selvatico/go-mocket"
)

func Test_QuotaManagementListCheckQuota(t *testing.T) {
	type fields struct {
		connectionFactory   *db.ConnectionFactory
		QuotaManagementList *quota_management.QuotaManagementListConfig
	}

	type args struct {
		instanceType types.DinosaurInstanceType
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "do not throw an error when instance limit control is disabled when checking eval instances",
			fields: fields{
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: false,
				},
			},
			args: args{
				instanceType: types.EVAL,
			},
			want: true,
		},
		{
			name: "return true when user is not part of the quota list and instance type is eval",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList:                  quota_management.RegisteredUsersListConfiguration{},
				},
			},
			args: args{
				instanceType: types.EVAL,
			},
			want: true,
		},
		{
			name: "return true when user is not part of the quota list and instance type is standard",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList:                  quota_management.RegisteredUsersListConfiguration{},
				},
			},
			args: args{
				instanceType: types.STANDARD,
			},
			want: false,
		},
		{
			name: "return true when user is part of the quota list as a service account and instance type is standard",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList: quota_management.RegisteredUsersListConfiguration{
						ServiceAccounts: quota_management.AccountList{
							quota_management.Account{
								Username:            "username",
								MaxAllowedInstances: 4,
							},
						},
					},
				},
			},
			args: args{
				instanceType: types.STANDARD,
			},
			want: true,
		},
		{
			name: "return true when user is part of the quota list under an organisation and instance type is standard",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList: quota_management.RegisteredUsersListConfiguration{
						Organisations: quota_management.OrganisationList{
							quota_management.Organisation{
								Id:                  "org-id",
								MaxAllowedInstances: 4,
								AnyUser:             true,
							},
						},
					},
				},
			},
			args: args{
				instanceType: types.STANDARD,
			},
			want: true,
		},
		{
			name: "return false when user is part of the quota list under an organisation and instance type is eval",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList: quota_management.RegisteredUsersListConfiguration{
						Organisations: quota_management.OrganisationList{
							quota_management.Organisation{
								Id:                  "org-id",
								MaxAllowedInstances: 4,
								AnyUser:             true,
							},
						},
					},
				},
			},
			args: args{
				instanceType: types.EVAL,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)

			factory := NewDefaultQuotaServiceFactory(nil, tt.fields.connectionFactory, tt.fields.QuotaManagementList)
			quotaService, _ := factory.GetQuotaService(api.QuotaManagementListQuotaType)
			dinosaur := &dbapi.DinosaurRequest{
				Owner:          "username",
				OrganisationId: "org-id",
			}
			allowed, _ := quotaService.CheckIfQuotaIsDefinedForInstanceType(dinosaur, tt.args.instanceType)
			gomega.Expect(tt.want).To(gomega.Equal(allowed))
		})
	}
}

func Test_QuotaManagementListReserveQuota(t *testing.T) {
	type fields struct {
		connectionFactory   *db.ConnectionFactory
		QuotaManagementList *quota_management.QuotaManagementListConfig
	}

	type args struct {
		instanceType types.DinosaurInstanceType
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr *errors.ServiceError
		setupFn func()
	}{
		{
			name: "do not return an error when instance limit control is disabled ",
			fields: fields{
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: false,
				},
			},
			args: args{
				instanceType: types.EVAL,
			},
			wantErr: nil,
		},
		{
			name: "return an error when the query db throws an error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList: quota_management.RegisteredUsersListConfiguration{
						ServiceAccounts: quota_management.AccountList{
							quota_management.Account{
								Username:            "username",
								MaxAllowedInstances: 4,
							},
						},
					},
				},
			},
			args: args{
				instanceType: types.EVAL,
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: errors.GeneralError("count failed from database"),
		},
		{
			name: "return an error when user in an organisation cannot create any more instances after exceeding allowed organisation limits",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList: quota_management.RegisteredUsersListConfiguration{
						Organisations: quota_management.OrganisationList{
							quota_management.Organisation{
								Id:                  "org-id",
								MaxAllowedInstances: 4,
								AnyUser:             true,
							},
						},
					},
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery(`SELECT count(1) FROM "dinosaur_requests" WHERE instance_type = $1 AND (organisation_id = $2)`).
					WithArgs(types.STANDARD.String(), "org-id").
					WithReply([]map[string]interface{}{{"count": "4"}})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: &errors.ServiceError{
				HttpCode: http.StatusForbidden,
				Reason:   "Organization 'org-id' has reached a maximum number of 4 allowed instances.",
				Code:     5,
			},
			args: args{
				instanceType: types.STANDARD,
			},
		},
		{
			name: "return an error when user in the quota list attempts to create an eval instance",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList: quota_management.RegisteredUsersListConfiguration{
						ServiceAccounts: quota_management.AccountList{
							quota_management.Account{
								Username:            "username",
								MaxAllowedInstances: 4,
							},
						},
					},
				},
			},
			args: args{
				instanceType: types.EVAL,
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery(`SELECT count(1) FROM "dinosaur_requests" WHERE instance_type = $1 AND owner = $2`).
					WithArgs(types.EVAL.String(), "username").
					WithReply([]map[string]interface{}{{"count": "0"}})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: errors.InsufficientQuotaError("Insufficient Quota"),
		},
		{
			name: "return an error when user is not allowed in their org and they cannot create any more instances eval instances after exceeding default allowed user limits",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList: quota_management.RegisteredUsersListConfiguration{
						Organisations: quota_management.OrganisationList{
							quota_management.Organisation{
								Id:                  "org-id",
								MaxAllowedInstances: 2,
								AnyUser:             false,
							},
						},
					},
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery(`SELECT count(1) FROM "dinosaur_requests" WHERE instance_type = $1 AND owner = $2`).
					WithArgs(types.EVAL.String(), "username").
					WithReply([]map[string]interface{}{{"count": "1"}})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: &errors.ServiceError{
				HttpCode: http.StatusForbidden,
				Reason:   "User 'username' has reached a maximum number of 1 allowed instances.",
				Code:     5,
			},
			args: args{
				instanceType: types.EVAL,
			},
		},
		{
			name: "does not return an error if user is within limits for user creating a standard instance",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList: quota_management.RegisteredUsersListConfiguration{
						Organisations: quota_management.OrganisationList{
							quota_management.Organisation{
								Id:                  "org-id",
								MaxAllowedInstances: 4,
								AnyUser:             true,
							},
						},
					},
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery(`SELECT count(1) FROM "dinosaur_requests" WHERE instance_type = $1 AND (organisation_id = $2)`).
					WithArgs(types.STANDARD.String(), "org-id").
					WithReply([]map[string]interface{}{{"count": "1"}})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			args: args{
				instanceType: types.STANDARD,
			},
			wantErr: nil,
		},
		{
			name: "do not return an error when user who's not in the quota list can eval instances",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery(`SELECT count(1) FROM "dinosaur_requests" WHERE instance_type = $1 AND owner = $2`).
					WithArgs(types.EVAL.String(), "username").
					WithReply([]map[string]interface{}{{"count": "0"}})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			args: args{
				instanceType: types.EVAL,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			if tt.setupFn != nil {
				tt.setupFn()
			}
			factory := NewDefaultQuotaServiceFactory(nil, tt.fields.connectionFactory, tt.fields.QuotaManagementList)
			quotaService, _ := factory.GetQuotaService(api.QuotaManagementListQuotaType)
			dinosaur := &dbapi.DinosaurRequest{
				Owner:          "username",
				OrganisationId: "org-id",
			}
			_, err := quotaService.ReserveQuota(dinosaur, tt.args.instanceType)
			gomega.Expect(tt.wantErr).To(gomega.Equal(err))
		})
	}
}
