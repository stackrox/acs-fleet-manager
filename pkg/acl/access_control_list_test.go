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
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RegisterTestingT(t)
			ok := tt.deniedAccounts.IsUserDenied(tt.arg)
			Expect(ok).To(Equal(tt.want))
		})
	}
}
