package services

import (
	"testing"

	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/pkg/errors"
	mocket "github.com/selvatico/go-mocket"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stretchr/testify/assert"
)

func TestDataMigration(t *testing.T) {
	tests := []struct {
		name        string
		expectedCnt int
		wantErr     bool
		setupFn     func()
	}{
		{
			name:        "migrate single org name",
			expectedCnt: 1,
			wantErr:     false,
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT *").
					WithReply([]map[string]interface{}{
						{"id": "dummy-id", "organisation_name": ""},
					})
			},
		},
		{
			name:        "migrate multiple org name",
			expectedCnt: 2,
			wantErr:     false,
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT *").
					WithReply([]map[string]interface{}{
						{"id": "dummy-id-1", "organisation_name": ""},
						{"id": "dummy-id-2", "organisation_name": ""},
					})
			},
		},
		{
			name:        "migrate no org names",
			expectedCnt: 0,
			wantErr:     false,
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT *").
					WithReply([]map[string]interface{}{})
			},
		},
		{
			name:        "migrate with error",
			expectedCnt: 0,
			wantErr:     true,
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT *").WithQueryException()

			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connectionFactory := db.NewMockConnectionFactory(nil)
			amsClient := ocm.ClientMock{GetOrganisationFromExternalIDFunc: func(externalID string) (*amsv1.Organization, error) {
				org, err := amsv1.NewOrganization().
					ID("12345678").
					Name("dummy-org").
					Build()
				return org, errors.Wrap(err, "failed to build organisation")
			}}
			tt.setupFn()

			dm := NewDataMigration(connectionFactory, &amsClient)
			cnt, err := dm.migrateOrganisationNames()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedCnt, cnt)
		})
	}
}
