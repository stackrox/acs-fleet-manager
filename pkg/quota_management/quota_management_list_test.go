package quota_management

import (
	. "github.com/onsi/gomega"
	"testing"
)

func Test_OrganisationList_GetById(t *testing.T) {
	t.Parallel()
	id := "org-id"
	type result struct {
		ok           bool
		organisation Organisation
	}

	tests := []struct {
		name string
		arg  string
		orgs OrganisationList
		want result
	}{
		{
			name: "return 'false' when organisation list is empty",
			arg:  id,
			orgs: OrganisationList{},
			want: result{
				ok:           false,
				organisation: Organisation{},
			},
		},
		{
			name: "return 'false' when organisation list does not contain the organisation for the given id",
			arg:  id,
			orgs: OrganisationList{
				Organisation{
					Id:              "org-1",
					RegisteredUsers: AccountList{},
				},
			},
			want: result{
				ok:           false,
				organisation: Organisation{},
			},
		},
		{
			name: "return 'true' and the corresponding organisation when organisation list contains the organisation for the given id",
			arg:  id,
			orgs: OrganisationList{
				Organisation{
					Id:              "org-2",
					RegisteredUsers: AccountList{Account{Username: "username"}},
				},
				Organisation{
					Id:              id,
					RegisteredUsers: AccountList{Account{Username: "user-1"}},
				},
			},
			want: result{
				ok: true,
				organisation: Organisation{
					Id:              id,
					RegisteredUsers: AccountList{Account{Username: "user-1"}},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RegisterTestingT(t)
			organisation, ok := tt.orgs.GetById(tt.arg)
			Expect(organisation).To(Equal(tt.want.organisation))
			Expect(ok).To(Equal(tt.want.ok))
		})
	}
}

func Test_Organisation_IsUserAllowed(t *testing.T) {
	t.Parallel()
	username := "username"
	tests := []struct {
		name string
		arg  string
		org  Organisation
		want bool
	}{
		{
			name: "return 'false' when quota list is empty",
			arg:  username,
			org:  Organisation{},
			want: false,
		},
		{
			name: "return 'false' when quota list does not contain the given username",
			arg:  username,
			org: Organisation{
				RegisteredUsers: AccountList{Account{Username: "user-1"}},
			},
			want: false,
		},
		{
			name: "return 'true' and the organisation quota list contains the given username",
			arg:  username,
			org: Organisation{
				RegisteredUsers: AccountList{
					Account{Username: "user-1"},
					Account{Username: username},
				},
			},
			want: true,
		},
		{
			name: "return 'true' if organisation is empty and all users are allowed",
			arg:  username,
			org: Organisation{
				RegisteredUsers: AccountList{},
				AnyUser:         true,
			},
			want: true,
		},
		{
			name: "return 'false' if organisation is empty and not all users are allowed",
			arg:  username,
			org: Organisation{
				RegisteredUsers: AccountList{},
				AnyUser:         false,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RegisterTestingT(t)
			ok := tt.org.IsUserRegistered(tt.arg)
			Expect(ok).To(Equal(tt.want))
		})
	}
}

func Test_Organisation_HasAllowedAccounts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		org  Organisation
		want bool
	}{
		{
			name: "return 'false' when the organisation quota list of users is empty",
			org: Organisation{
				RegisteredUsers: AccountList{},
			},
			want: false,
		},
		{
			name: "return 'true' when the organisation contains a non empty quota list of users",
			org: Organisation{
				RegisteredUsers: AccountList{
					Account{Username: "username-1"},
					Account{Username: "username-2"},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RegisterTestingT(t)
			ok := tt.org.HasUsersRegistered()
			Expect(ok).To(Equal(tt.want))
		})
	}
}

func Test_AllowedAccounts_GetByUsername(t *testing.T) {
	t.Parallel()
	username := "username"
	type result struct {
		found          bool
		allowedAccount Account
	}
	tests := []struct {
		name            string
		arg             string
		allowedAccounts AccountList
		want            result
	}{
		{
			name:            "return 'false' when allowed users is empty",
			arg:             username,
			allowedAccounts: AccountList{},
			want: result{
				found:          false,
				allowedAccount: Account{},
			},
		},
		{
			name:            "return 'false' when allowed users does not contain the given username",
			arg:             username,
			allowedAccounts: AccountList{Account{Username: "user-1"}},
			want: result{
				found:          false,
				allowedAccount: Account{},
			},
		},
		{
			name: "return 'true' and the allowed user when when allowed users contains the given username",
			arg:  username,
			allowedAccounts: AccountList{
				Account{Username: "user-1"},
				Account{Username: username},
			},
			want: result{
				found:          true,
				allowedAccount: Account{Username: username},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RegisterTestingT(t)
			user, found := tt.allowedAccounts.GetByUsername(tt.arg)
			Expect(user).To(Equal(tt.want.allowedAccount))
			Expect(found).To(Equal(tt.want.found))
		})
	}
}

func Test_AllowedAccount_IsInstanceCountWithinLimit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		count int
		item  QuotaManagementListItem
		want  bool
	}{
		{
			name:  "return 'false' when count is above or equal max allowed instances for the given organisation",
			count: 3,
			item:  Organisation{MaxAllowedInstances: 2},
			want:  false,
		},
		{
			name:  "return 'true' when count is below max allowed instances for the given organisation",
			count: 1,
			item:  Organisation{MaxAllowedInstances: 2},
			want:  true,
		},
		{
			name:  "return 'false' when count is above or equal default value (1) of max allowed instances for the given organisation without max allowed instances set",
			count: 1,
			item:  Organisation{},
			want:  false,
		},
		{
			name:  "return 'true' when count is below the default value (1) of max allowed instances for the given organisation without max allowed instances set",
			count: 0,
			item:  Organisation{},
			want:  true,
		},
		{
			name:  "return 'false' when count is above or equal max allowed instances for the given account",
			count: 3,
			item:  Account{MaxAllowedInstances: 2},
			want:  false,
		},
		{
			name:  "return 'true' when count is below max allowed instances for the given account",
			count: 1,
			item:  Account{MaxAllowedInstances: 2},
			want:  true,
		},
		{
			name:  "return 'false' when count is above or equal default value (1) of max allowed instances for the given account without max allowed instances set",
			count: 1,
			item:  Account{},
			want:  false,
		},
		{
			name:  "return 'true' when count is below the default value (1) of max allowed instances for the given account without max allowed instances set",
			count: 0,
			item:  Account{},
			want:  true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RegisterTestingT(t)
			ok := tt.item.IsInstanceCountWithinLimit(tt.count)
			Expect(ok).To(Equal(tt.want))
		})
	}
}

func Test_AllowedAccount_GetMaxAllowedInstances(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		allowedAccount Account
		want           int
	}{
		{
			name:           "return max allowed instances of the user when its value is given",
			allowedAccount: Account{MaxAllowedInstances: 2},
			want:           2,
		},
		{
			name:           "return default value (1) of max allowed instances when max allowed instance per user is not set",
			allowedAccount: Account{},
			want:           1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RegisterTestingT(t)
			ok := tt.allowedAccount.GetMaxAllowedInstances()
			Expect(ok).To(Equal(tt.want))
		})
	}
}
