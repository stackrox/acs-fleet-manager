package services

import (
	"context"
	"database/sql/driver"
	"reflect"
	"testing"
	"time"

	mocket "github.com/selvatico/go-mocket"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/centrals/types"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/converters"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	JwtKeyFile = "test/support/jwt_private_key.pem"
	JwtCAFile  = "test/support/jwt_ca.pem"
)

var (
	testCentralRequestRegion   = "us-east-1"
	testCentralRequestProvider = "aws"
	testCentralRequestName     = "test-cluster"
	testClusterID              = "test-cluster-id"
	testID                     = "test"
	testUser                   = "test-user"
)

// build a test central request
func buildCentralRequest(modifyFn func(centralRequest *dbapi.CentralRequest)) *dbapi.CentralRequest {
	centralRequest := &dbapi.CentralRequest{
		Meta: api.Meta{
			ID:        testID,
			DeletedAt: gorm.DeletedAt{Valid: true},
		},
		Region:        testCentralRequestRegion,
		ClusterID:     testClusterID,
		CloudProvider: testCentralRequestProvider,
		Name:          testCentralRequestName,
		MultiAZ:       false,
		Owner:         testUser,
	}
	if modifyFn != nil {
		modifyFn(centralRequest)
	}
	return centralRequest
}

// This test should act as a "golden" test to describe the general testing approach taken in the service, for people
// onboarding into development of the service.
func Test_centralService_Get(t *testing.T) {
	// fields are the variables on the struct that we're testing, in this case centralService
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	// args are the variables that will be provided to the function we're testing, in this case it's just the id we
	// pass to centralService.PrepareCentralRequest
	type args struct {
		ctx context.Context
		id  string
	}

	authHelper, err := auth.NewAuthHelper(JwtKeyFile, JwtCAFile, "")
	if err != nil {
		t.Fatalf("failed to create auth helper: %s", err.Error())
	}
	account, err := authHelper.NewAccount(testUser, "", "", "")
	if err != nil {
		t.Fatal("failed to build a new account")
	}

	jwt, err := authHelper.CreateJWTWithClaims(account, nil)
	if err != nil {
		t.Fatalf("failed to create jwt: %s", err.Error())
	}
	ctx := context.TODO()
	authenticatedCtx := auth.SetTokenInContext(ctx, jwt)

	// we define tests as list of structs that contain inputs and expected outputs
	// this means we can execute the same logic on each test struct, and makes adding new tests simple as we only need
	// to provide a new struct to the list instead of defining an entirely new test
	tests := []struct {
		// name is just a description of the test
		name   string
		fields fields
		args   args
		// want (there can be more than one) is the outputs that we expect, they can be compared after the test
		// function has been executed
		want *dbapi.CentralRequest
		// wantErr is similar to want, but instead of testing the actual returned error, we're just testing than any
		// error has been returned
		wantErr bool
		// setupFn will be called before each test and allows mocket setup to be performed
		setupFn func()
	}{
		// below is a single test case, we define each of the fields that we care about from the anonymous test struct
		// above
		{
			name: "error when id is undefined",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				ctx: context.TODO(),
				id:  "",
			},
			wantErr: true,
		},
		{
			name: "error when sql where query fails",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				ctx: authenticatedCtx,
				id:  testID,
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithQueryException()
			},
		},
		{
			name: "successful output",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				ctx: authenticatedCtx,
				id:  testID,
			},
			want: buildCentralRequest(nil),
			setupFn: func() {
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "central_requests" WHERE id = $1 AND owner = $2 AND "central_requests"."deleted_at" IS NULL ORDER BY "central_requests"."id" LIMIT $3`).
					WithArgs(testID, testUser, int64(1)).
					WithReply(converters.ConvertCentralRequest(buildCentralRequest(nil)))
			},
		},
	}
	// we loop through each test case defined in the list above and start a new test invocation, using the testing
	// t.Run function
	for _, tt := range tests {
		// tt now contains our test case, we can use the 'fields' to construct the struct that we want to test and the
		// 'args' to pass to the function we want to test
		t.Run(tt.name, func(t *testing.T) {
			// invoke any pre-req logic if needed
			if tt.setupFn != nil {
				tt.setupFn()
			}
			// we're testing the centralService struct, so use the 'fields' to create one
			k := &centralService{
				connectionFactory: tt.fields.connectionFactory,
			}
			// we're testing the centralService.Get function so use the 'args' to provide arguments to the function
			got, err := k.Get(tt.args.ctx, tt.args.id)
			// in our test case we used 'wantErr' to define if we expect and error to be returned from the function or
			// not, now we test that expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// in our test case we used 'want' to define the output api.CentralRequest that we expect to be returned, we
			// can use reflect.DeepEqual to compare the actual struct with the expected struct
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_centralService_DeprovisionExpiredCentralsQuery(t *testing.T) {
	k := &centralService{
		connectionFactory: db.NewMockConnectionFactory(nil),
		centralConfig: &config.CentralConfig{
			CentralLifespan: config.NewCentralLifespanConfig(),
		},
	}

	m := mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "central_requests" ` +
		`SET "deletion_timestamp"=$1,"status"=$2,"updated_at"=$3 WHERE ` +
		`(expired_at IS NOT NULL AND expired_at < $4 OR instance_type = $5 AND created_at <= $6 ` +
		`AND (expired_at IS NOT NULL AND expired_at < $7 OR instance_type = $8 AND created_at <= $9) ` +
		`AND status NOT IN ($10,$11)) AND "central_requests"."deleted_at" IS NULL`).
		OneTime()

	svcErr := k.DeprovisionExpiredCentrals()
	assert.Nil(t, svcErr)
	assert.True(t, m.Triggered)

	m = mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "central_requests" ` +
		`SET "deletion_timestamp"=$1,"status"=$2,"updated_at"=$3 WHERE ` +
		`expired_at IS NOT NULL AND expired_at < $4 ` +
		`AND status NOT IN ($5,$6) AND "central_requests"."deleted_at" IS NULL`).
		OneTime()
	k.centralConfig.CentralLifespan.EnableDeletionOfExpiredCentral = false
	svcErr = k.DeprovisionExpiredCentrals()
	assert.Nil(t, svcErr)
	assert.True(t, m.Triggered)
}

func Test_centralService_RestoreExpiredCentrals(t *testing.T) {
	dbConnectionFactory := db.NewMockConnectionFactory(nil)

	centralService := &centralService{
		connectionFactory: dbConnectionFactory,
		centralConfig: &config.CentralConfig{
			CentralLifespan:            config.NewCentralLifespanConfig(),
			CentralRetentionPeriodDays: 2,
		},
	}

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	m := mocket.Catcher.Reset().NewMock()
	selectQuery := m.WithQuery(`SELECT`).
		WithReply([]map[string]interface{}{{
			"id":         "test-id",
			"deleted_at": yesterday,
			"expired_at": yesterday,
		}}).OneTime()

	m1 := mocket.Catcher.NewMock()
	expiredChecked := false
	updateQuery := m1.WithQuery(`UPDATE`).WithCallback(
		func(s string, nv []driver.NamedValue) {
			expiredAt, _ := (nv[10].Value).(*time.Time)
			assert.Nil(t, expiredAt)
			assert.Equal(t, "test-id", nv[11].Value)
			expiredChecked = true
		})
	svcErr := centralService.Restore(context.Background(), "test-id")
	assert.Nil(t, svcErr)
	assert.True(t, selectQuery.Triggered)
	assert.True(t, updateQuery.Triggered)
	assert.True(t, expiredChecked)
}

func Test_centralService_ChangeBillingParameters(t *testing.T) {
	quotaService := &QuotaServiceMock{
		HasQuotaAllowanceFunc: func(central *dbapi.CentralRequest, instanceType types.CentralInstanceType) (bool, *errors.ServiceError) {
			return true, nil
		},
		ReserveQuotaFunc: func(ctx context.Context, central *dbapi.CentralRequest, _ string, _ string) (string, *errors.ServiceError) {
			return central.SubscriptionID, nil
		},
	}
	quotaServiceFactory := &QuotaServiceFactoryMock{
		GetQuotaServiceFunc: func(quotaType api.QuotaType) (QuotaService, *errors.ServiceError) {
			return quotaService, nil
		},
	}
	k := &centralService{
		centralConfig:       config.NewCentralConfig(),
		connectionFactory:   db.NewMockConnectionFactory(nil),
		quotaServiceFactory: quotaServiceFactory,
	}
	central := buildCentralRequest(func(centralRequest *dbapi.CentralRequest) {
		centralRequest.QuotaType = "standard"
		centralRequest.OrganisationID = "original org ID"
		centralRequest.CloudProvider = ""
		centralRequest.CloudAccountID = ""
		centralRequest.SubscriptionID = "original subscription ID"
	})

	catcher := mocket.Catcher.Reset()
	m0 := catcher.NewMock().WithQuery(`SELECT * FROM "central_requests" `+
		`WHERE id = $1 AND "central_requests"."deleted_at" IS NULL `+
		`ORDER BY "central_requests"."id" LIMIT $2`).
		OneTime().WithArgs(testID, int64(1)).
		WithReply(converters.ConvertCentralRequest(central))
	m1 := catcher.NewMock().WithQuery(`UPDATE "central_requests" ` +
		`SET "updated_at"=$1,"deleted_at"=$2,"region"=$3,"cluster_id"=$4,` +
		`"cloud_provider"=$5,"cloud_account_id"=$6,"name"=$7,"subscription_id"=$8,"owner"=$9 ` +
		`WHERE status not IN ($10,$11) AND "central_requests"."deleted_at" IS NULL AND "id" = $12`).
		OneTime()

	svcErr := k.ChangeBillingParameters(context.Background(), central.ID, "marketplace", "aws_account_id", "aws", "")
	assert.Nil(t, svcErr)

	assert.True(t, m0.Triggered)
	assert.True(t, m1.Triggered)

	qsfCalls := quotaServiceFactory.GetQuotaServiceCalls()
	require.Len(t, qsfCalls, 1)

	reserveQuotaCalls := quotaService.ReserveQuotaCalls()
	require.Len(t, reserveQuotaCalls, 1)
	assert.Equal(t, testID, reserveQuotaCalls[0].Central.ID)

	deleteQuotaCalls := quotaService.DeleteQuotaCalls()
	require.Len(t, deleteQuotaCalls, 0)
}

func Test_centralService_ChangeSubscription(t *testing.T) {
	service := &centralService{
		connectionFactory: db.NewMockConnectionFactory(nil),
	}
	central := buildCentralRequest(func(centralRequest *dbapi.CentralRequest) {
		centralRequest.CloudProvider = ""
		centralRequest.CloudAccountID = ""
		centralRequest.SubscriptionID = "original_subscription_id"
	})

	catcher := mocket.Catcher.Reset()
	q0 := catcher.NewMock().WithQuery(`SELECT * FROM "central_requests" `+
		`WHERE id = $1 AND "central_requests"."deleted_at" IS NULL `+
		`ORDER BY "central_requests"."id" LIMIT $2`).
		OneTime().WithArgs(testID, int64(1)).
		WithReply(converters.ConvertCentralRequest(central))
	q1 := catcher.NewMock().WithQuery(`UPDATE "central_requests" ` +
		`SET "updated_at"=$1,"deleted_at"=$2,"region"=$3,"cluster_id"=$4,` +
		`"cloud_provider"=$5,"cloud_account_id"=$6,"name"=$7,"subscription_id"=$8,"owner"=$9 ` +
		`WHERE status not IN ($10,$11) AND "central_requests"."deleted_at" IS NULL AND "id" = $12`).
		OneTime()

	svcErr := service.ChangeSubscription(context.Background(), central.ID, "aws_account_id", "aws", "new_subscription_id")
	assert.Nil(t, svcErr)

	assert.True(t, q0.Triggered)
	assert.True(t, q1.Triggered)

}
