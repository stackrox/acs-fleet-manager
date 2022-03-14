package config

import (
	"reflect"
	"testing"

	"github.com/stackrox/acs-fleet-manager/pkg/quota_management"

	. "github.com/onsi/gomega"
)

func Test_configService_GetDefaultProvider(t *testing.T) {
	type fields struct {
		providersConfig ProviderConfig
	}
	tests := []struct {
		name    string
		fields  fields
		want    Provider
		wantErr bool
	}{
		{
			name: "error when no default provider found",
			fields: fields{
				providersConfig: ProviderConfig{
					ProvidersConfig: ProviderConfiguration{
						SupportedProviders: ProviderList{},
					},
				},
			},
			wantErr: true,
			want:    Provider{},
		},
		{
			name: "success when default provider found",
			fields: fields{
				providersConfig: ProviderConfig{
					ProvidersConfig: ProviderConfiguration{
						SupportedProviders: ProviderList{
							Provider{
								Name:    "test",
								Default: true,
							},
						},
					},
				},
			},
			want: Provider{
				Name:    "test",
				Default: true,
			},
		},
		{
			name: "first default returned when multiple defaults specified",
			fields: fields{
				providersConfig: ProviderConfig{
					ProvidersConfig: ProviderConfiguration{
						SupportedProviders: ProviderList{
							Provider{
								Name:    "test1",
								Default: true,
							},
							Provider{
								Name:    "test2",
								Default: true,
							},
						},
					},
				},
			},
			want: Provider{
				Name:    "test1",
				Default: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.fields.providersConfig
			got, err := c.ProvidersConfig.SupportedProviders.GetDefault()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDefaultProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDefaultProvider() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_configService_GetDefaultRegionForProvider(t *testing.T) {
	type args struct {
		provider Provider
	}
	tests := []struct {
		name    string
		args    args
		want    Region
		wantErr bool
	}{
		{
			name: "error when no default region found",
			args: args{
				provider: Provider{
					Regions: RegionList{},
				},
			},
			want:    Region{},
			wantErr: true,
		},
		{
			name: "success when default region found",
			args: args{
				provider: Provider{
					Regions: RegionList{
						Region{
							Name:    "test",
							Default: true,
						},
					},
				},
			},
			want: Region{
				Name:    "test",
				Default: true,
			},
		},
		{
			name: "first default returned when multiple defaults specified",
			args: args{
				provider: Provider{
					Regions: RegionList{
						Region{
							Name:    "test1",
							Default: true,
						},
						Region{
							Name:    "test2",
							Default: true,
						},
					},
				},
			},
			want: Region{
				Name:    "test1",
				Default: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.args.provider.GetDefaultRegion()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDefaultRegionForProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDefaultRegionForProvider() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_configService_GetSupportedProviders(t *testing.T) {
	type fields struct {
		providersConfig ProviderConfig
	}
	tests := []struct {
		name   string
		fields fields
		want   ProviderList
	}{
		{
			name: "successful get",
			fields: fields{
				providersConfig: ProviderConfig{
					ProvidersConfig: ProviderConfiguration{
						SupportedProviders: ProviderList{
							Provider{
								Name: "test",
							},
						},
					},
				},
			},
			want: ProviderList{
				Provider{
					Name: "test",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.fields.providersConfig
			if got := c.ProvidersConfig.SupportedProviders; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSupportedProviders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_configService_IsProviderSupported(t *testing.T) {
	type fields struct {
		providersConfig ProviderConfig
	}
	type args struct {
		providerName string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "false when provider not in supported list",
			fields: fields{
				providersConfig: ProviderConfig{
					ProvidersConfig: ProviderConfiguration{
						SupportedProviders: ProviderList{},
					},
				},
			},
			args: args{
				providerName: "test",
			},
			want: false,
		},
		{
			name: "true when provider in supported list",
			fields: fields{
				providersConfig: ProviderConfig{
					ProvidersConfig: ProviderConfiguration{
						SupportedProviders: ProviderList{
							Provider{
								Name: "test",
							},
						},
					},
				},
			},
			args: args{
				providerName: "test",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.fields.providersConfig
			if _, got := c.ProvidersConfig.SupportedProviders.GetByName(tt.args.providerName); got != tt.want {
				t.Errorf("IsProviderSupported() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_configService_GetOrganisationById(t *testing.T) {
	type result struct {
		found        bool
		organisation quota_management.Organisation
	}

	tests := []struct {
		name                string
		QuotaManagementList *quota_management.QuotaManagementListConfig
		arg                 string
		want                result
	}{
		{
			name: "return 'false' when organisation does not exist in the allowed list",
			arg:  "some-id",
			QuotaManagementList: &quota_management.QuotaManagementListConfig{
				QuotaList: quota_management.RegisteredUsersListConfiguration{
					Organisations: quota_management.OrganisationList{
						quota_management.Organisation{
							Id: "different-id",
						},
					},
				},
			},
			want: result{
				found:        false,
				organisation: quota_management.Organisation{},
			},
		},
		{
			name: "return 'true' when organisation exists in the allowed list",
			arg:  "some-id",
			QuotaManagementList: &quota_management.QuotaManagementListConfig{
				QuotaList: quota_management.RegisteredUsersListConfiguration{
					Organisations: quota_management.OrganisationList{
						quota_management.Organisation{
							Id: "some-id",
						},
					},
				},
			},
			want: result{
				found: true,
				organisation: quota_management.Organisation{
					Id: "some-id",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			org, found := tt.QuotaManagementList.QuotaList.Organisations.GetById(tt.arg)
			Expect(org).To(Equal(tt.want.organisation))
			Expect(found).To(Equal(tt.want.found))
		})
	}
}

func Test_configService_GetAllowedAccountByUsernameAndOrgId(t *testing.T) {
	type args struct {
		username string
		orgId    string
	}

	type result struct {
		AllowedAccount quota_management.Account
		found          bool
	}

	organisation := quota_management.Organisation{
		Id: "some-id",
		RegisteredUsers: quota_management.AccountList{
			quota_management.Account{Username: "username-0"},
			quota_management.Account{Username: "username-1"},
		},
	}

	tests := []struct {
		name                string
		arg                 args
		want                result
		QuotaManagementList *quota_management.QuotaManagementListConfig
	}{
		{
			name: "return 'true' and the found user when organisation contains the user",
			arg: args{
				username: "username-1",
				orgId:    organisation.Id,
			},
			QuotaManagementList: &quota_management.QuotaManagementListConfig{
				QuotaList: quota_management.RegisteredUsersListConfiguration{
					Organisations: quota_management.OrganisationList{
						organisation,
					},
				},
			},
			want: result{
				found:          true,
				AllowedAccount: quota_management.Account{Username: "username-1"},
			},
		},
		{
			name: "return 'true' and the user when user is not among the listed organisation but is contained in list of allowed service accounts",
			arg: args{
				username: "username-10",
				orgId:    organisation.Id,
			},
			QuotaManagementList: &quota_management.QuotaManagementListConfig{
				QuotaList: quota_management.RegisteredUsersListConfiguration{
					Organisations: quota_management.OrganisationList{
						organisation,
					},
					ServiceAccounts: quota_management.AccountList{
						quota_management.Account{Username: "username-0"},
						quota_management.Account{Username: "username-10"},
						quota_management.Account{Username: "username-3"},
					},
				},
			},
			want: result{
				found:          true,
				AllowedAccount: quota_management.Account{Username: "username-10"},
			},
		},
		{
			name: "return 'false' when user is not among the listed organisation and in list of allowed service accounts",
			arg: args{
				username: "username-10",
				orgId:    "some-org-id",
			},
			QuotaManagementList: &quota_management.QuotaManagementListConfig{
				QuotaList: quota_management.RegisteredUsersListConfiguration{
					Organisations: quota_management.OrganisationList{
						organisation,
					},
					ServiceAccounts: quota_management.AccountList{
						quota_management.Account{Username: "username-0"},
						quota_management.Account{Username: "username-3"},
					},
				},
			},
			want: result{
				found: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			user, ok := tt.QuotaManagementList.GetAllowedAccountByUsernameAndOrgId(tt.arg.username, tt.arg.orgId)
			Expect(user).To(Equal(tt.want.AllowedAccount))
			Expect(ok).To(Equal(tt.want.found))
		})
	}
}

func Test_configService_GetServiceAccountByUsername(t *testing.T) {
	type args struct {
		username string
	}

	type result struct {
		AllowedAccount quota_management.Account
		found          bool
	}

	organisation := quota_management.Organisation{
		Id: "some-id",
		RegisteredUsers: quota_management.AccountList{
			quota_management.Account{Username: "username-0"},
			quota_management.Account{Username: "username-1"},
		},
	}

	tests := []struct {
		name                string
		arg                 args
		want                result
		QuotaManagementList *quota_management.QuotaManagementListConfig
	}{
		{
			name: "return 'true' and the user when user is contained in list of allowed service accounts",
			arg: args{
				username: "username-10",
			},
			QuotaManagementList: &quota_management.QuotaManagementListConfig{
				QuotaList: quota_management.RegisteredUsersListConfiguration{
					Organisations: quota_management.OrganisationList{
						organisation,
					},
					ServiceAccounts: quota_management.AccountList{
						quota_management.Account{Username: "username-0"},
						quota_management.Account{Username: "username-10"},
						quota_management.Account{Username: "username-3"},
					},
				},
			},
			want: result{
				found:          true,
				AllowedAccount: quota_management.Account{Username: "username-10"},
			},
		},
		{
			name: "return 'false' when user is not in the list of allowed service accounts",
			arg: args{
				username: "username-10",
			},
			QuotaManagementList: &quota_management.QuotaManagementListConfig{
				QuotaList: quota_management.RegisteredUsersListConfiguration{
					Organisations: quota_management.OrganisationList{
						organisation,
					},
					ServiceAccounts: quota_management.AccountList{
						quota_management.Account{Username: "username-0"},
						quota_management.Account{Username: "username-3"},
					},
				},
			},
			want: result{
				found: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			user, ok := tt.QuotaManagementList.QuotaList.ServiceAccounts.GetByUsername(tt.arg.username)
			Expect(user).To(Equal(tt.want.AllowedAccount))
			Expect(ok).To(Equal(tt.want.found))
		})
	}
}

func Test_configService_Validate(t *testing.T) {
	type fields struct {
		providersConfig ProviderConfig
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "error when no default provider provided",
			fields: fields{
				providersConfig: ProviderConfig{
					ProvidersConfig: ProviderConfiguration{
						SupportedProviders: ProviderList{
							Provider{
								Name:    "test",
								Default: false,
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error when no default region in default provider",
			fields: fields{
				providersConfig: ProviderConfig{
					ProvidersConfig: ProviderConfiguration{
						SupportedProviders: ProviderList{
							Provider{
								Name:    "test",
								Default: true,
								Regions: RegionList{
									Region{
										Name:    "test",
										Default: false,
									},
								},
							},
							Provider{
								Name: "test",
								Regions: RegionList{
									Region{
										Name:    "test",
										Default: true,
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error when multiple default providers provided",
			fields: fields{
				providersConfig: ProviderConfig{
					ProvidersConfig: ProviderConfiguration{
						SupportedProviders: ProviderList{
							Provider{
								Name:    "test1",
								Default: true,
							},
							Provider{
								Name:    "test2",
								Default: true,
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error when multiple default regions in default provider",
			fields: fields{
				providersConfig: ProviderConfig{
					ProvidersConfig: ProviderConfiguration{
						SupportedProviders: ProviderList{
							Provider{
								Name:    "test",
								Default: true,
								Regions: RegionList{
									Region{
										Name:    "test1",
										Default: true,
									},
									Region{
										Name:    "test2",
										Default: true,
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "success when default provider and region provided",
			fields: fields{
				providersConfig: ProviderConfig{
					ProvidersConfig: ProviderConfiguration{
						SupportedProviders: ProviderList{
							Provider{
								Name:    "test",
								Default: true,
								Regions: RegionList{
									Region{
										Name:    "test1",
										Default: true,
									},
									Region{
										Name:    "test2",
										Default: false,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.fields.providersConfig
			if err := c.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_configService_validateProvider(t *testing.T) {
	type args struct {
		provider Provider
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "error when no default region in provider",
			args: args{
				provider: Provider{
					Name:    "test",
					Default: false,
				},
			},
			wantErr: true,
		},
		{
			name: "error when more than one default region in provider",
			args: args{
				provider: Provider{
					Name: "test",
					Regions: RegionList{
						Region{
							Name:    "test1",
							Default: true,
						},
						Region{
							Name:    "test2",
							Default: true,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "success when default region provided",
			args: args{
				provider: Provider{
					Name: "test",
					Regions: RegionList{
						Region{
							Name:    "test",
							Default: true,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.args.provider.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("validateProvider() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
