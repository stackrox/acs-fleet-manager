package acl

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test_DeniedUsers_IsUserDenied(t *testing.T) {
	t.Parallel()
	username := "user-name"
	tests := []struct {
		name           string
		arg            string
		deniedAccounts DeniedUsers
		want           bool
	}{
		{
			name:           "return 'false' when denied accounts list is empty",
			arg:            username,
			deniedAccounts: DeniedUsers{},
			want:           false,
		},
		{
			name:           "return 'false' when denied accounts list does not contain the given username",
			arg:            username,
			deniedAccounts: DeniedUsers{"user1", "user2"},
			want:           false,
		},
		{
			name:           "return 'true' when denied accounts list contains the given username",
			arg:            username,
			deniedAccounts: DeniedUsers{"user1", "user-name", "user2"},
			want:           true,
		},
	}
	RegisterTestingT(t)
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ok := tt.deniedAccounts.IsUserDenied(tt.arg)
			Expect(ok).To(Equal(tt.want))
		})
	}
}

func Test_ReadDenyListConfigFile(t *testing.T) {
	type args struct {
		file        string
		deniedUsers *DeniedUsers
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "should return an error when attempting to read deny list with non-existent file",
			args: args{
				file:        "non-existent path",
				deniedUsers: &DeniedUsers{},
			},
			wantErr: true,
		},
		{
			name: "should not return an error when attempting to read deny list with non-existent file",
			args: args{
				file:        "config/deny-list-configuration.yaml",
				deniedUsers: &DeniedUsers{},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := readDenyListConfigFile(tt.args.file, tt.args.deniedUsers)
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_ReadFiles(t *testing.T) {
	type args struct {
		file           string
		deniedUsers    *DeniedUsers
		enableDenyList bool
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "should not return an error when deny list is enabled and deny list config file path is valid",
			args: args{
				file:           "config/deny-list-configuration.yaml",
				deniedUsers:    &DeniedUsers{},
				enableDenyList: true,
			},
			wantErr: false,
		},
		{
			name: "should return an error when deny list is enabled and config file path is invalid",
			args: args{
				file:           "invalid-path",
				deniedUsers:    &DeniedUsers{},
				enableDenyList: true,
			},
			wantErr: true,
		},
		{
			name: "should not return an error when deny list is disabled",
			args: args{
				file:           "config/deny-list-configuration.yaml",
				deniedUsers:    &DeniedUsers{},
				enableDenyList: false,
			},
			wantErr: false,
		},
		{
			name: "should not return an error when deny list is disabled and deny list config file path is invalid",
			args: args{
				file:           "config/deny-list-configuration.yaml",
				deniedUsers:    &DeniedUsers{},
				enableDenyList: false,
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aclConfig := NewAccessControlListConfig()
			aclConfig.DenyList = *tt.args.deniedUsers
			aclConfig.EnableDenyList = tt.args.enableDenyList
			aclConfig.DenyListConfigFile = tt.args.file
			err := aclConfig.ReadFiles()
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}
