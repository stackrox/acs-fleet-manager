// Package test ...
package test

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/config"
	"github.com/stackrox/acs-fleet-manager/pkg/shared/testutils"

	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	ocm "github.com/stackrox/acs-fleet-manager/pkg/client/ocm/impl"
	"github.com/stackrox/acs-fleet-manager/pkg/server"

	"github.com/stackrox/acs-fleet-manager/internal/central/compat"

	"github.com/goava/di"
	"github.com/golang/glog"
	gm "github.com/onsi/gomega"
	"github.com/spf13/pflag"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"

	"github.com/bxcodec/faker/v3"
	"github.com/golang-jwt/jwt/v4"
	amv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/rs/xid"
	adminprivate "github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
	"github.com/stackrox/acs-fleet-manager/test/mocks"
)

const (
	jwtKeyFile         = "test/support/jwt_private_key.pem"
	jwtCAFile          = "test/support/jwt_ca.pem"
	dataplaneIssuerURI = "https://dataplane.issuer.test.local"
)

// TODO jwk mock server needs to be refactored out of the helper and into the testing environment
// var jwkURL string

// TimeFunc defines a way to get a new Time instance common to the entire test suite.
// Aria's environment has Virtual Time that may not be actual time. We compensate
// by synchronizing on a common time func attached to the test harness.
type TimeFunc func() time.Time

// Helper ...
type Helper struct {
	AuthHelper    *auth.AuthHelper
	JWTPrivateKey *rsa.PrivateKey
	JWTCA         *rsa.PublicKey
	T             *testing.T
	Env           *environments.Env
}

// NewHelperWithHooks will init the Helper and start the server, and it allows to customize the configurations of the server via the hook.
// The startHook will be invoked after the environments.Env is created but before the api server is started, which will allow caller to change configurations.
// The startHook can should be a function and can optionally have type arguments that can be injected from the configuration container.
func NewHelperWithHooks(t *testing.T, httpServer *httptest.Server, configurationHook interface{}, envProviders ...di.Option) (*Helper, func()) {

	// Register the test with gomega
	gm.RegisterTestingT(t)

	// Manually set environment name, ignoring environment variables
	validTestEnv := false
	envName := environments.GetEnvironmentStrFromEnv()
	for _, testEnv := range []string{environments.TestingEnv, environments.IntegrationEnv, environments.DevelopmentEnv} {
		if envName == testEnv {
			validTestEnv = true
			break
		}
	}
	if !validTestEnv {
		fmt.Println("OCM_ENV environment variable not set to a valid test environment, using default testing environment")
		envName = environments.TestingEnv
	}
	h := &Helper{
		T: t,
	}

	if configurationHook != nil {
		envProviders = append(envProviders, di.ProvideValue(environments.BeforeCreateServicesHook{
			Func: configurationHook,
		}))
	}

	var err error
	env, err := environments.New(envName, envProviders...)
	if err != nil {
		glog.Fatalf("error initializing: %v", err)
	}
	h.Env = env

	parseCommandLineFlags(env)

	var ocmConfig *ocm.OCMConfig
	var iamConfig *iam.IAMConfig
	var centralConfig *config.CentralConfig

	env.MustResolveAll(&ocmConfig, &iamConfig, &centralConfig)

	// Create a new helper
	authHelper, err := auth.NewAuthHelper(jwtKeyFile, jwtCAFile, iamConfig.RedhatSSORealm.ValidIssuerURI)
	if err != nil {
		t.Fatalf("failed to create a new auth helper %s", err.Error())
	}
	h.JWTPrivateKey = authHelper.JWTPrivateKey // pragma: allowlist secret
	h.JWTCA = authHelper.JWTCA
	h.AuthHelper = authHelper

	// Set server if provided
	if httpServer != nil && ocmConfig.MockMode == ocm.MockModeEmulateServer {
		workers.DefaultRepeatInterval = 1 * time.Second
		fmt.Printf("Setting OCM base URL to %s\n", httpServer.URL)
		ocmConfig.BaseURL = httpServer.URL
		ocmConfig.AmsURL = httpServer.URL
	}

	jwkURL, stopJWKMockServer := h.StartJWKCertServerMock()
	iamConfig.JwksURL = jwkURL
	iamConfig.DataPlaneOIDCIssuers = &iam.OIDCIssuers{
		URIs: []string{
			dataplaneIssuerURI,
		},
		JWKSURIs: []string{
			jwkURL,
			"https://dummy", // append https endpoint to the end to exploit a bug in ocm-sdk that allows to use insecure http endpoints.
		},
	}

	file := testutils.CreateNonEmptyFile(t)
	defer os.Remove(file.Name())
	centralConfig.CentralIDPClientSecretFile = file.Name()
	centralConfig.CentralIDPClientID = "mock-client-id"

	// the configuration hook might set config options that influence which config files are loaded,
	// by env.LoadConfig()
	if configurationHook != nil {
		env.MustInvoke(configurationHook)
	}

	// loads the config files and create the services...
	err = env.CreateServices()
	if err != nil {
		glog.Fatalf("Unable to initialize testing environment: %s", err.Error())
	}

	h.CleanDB()
	h.ResetDB()
	env.Start()
	return h, buildTeardownHelperFn(
		env.Stop,
		h.CleanDB,
		metrics.Reset,
		stopJWKMockServer,
		env.Cleanup)
}

func parseCommandLineFlags(env *environments.Env) {
	commandLine := pflag.NewFlagSet("test", pflag.PanicOnError)
	err := env.AddFlags(commandLine)
	if err != nil {
		glog.Fatalf("Unable to add environment flags: %s", err.Error())
	}
	if logLevel := os.Getenv("LOGLEVEL"); logLevel != "" {
		glog.Infof("Using custom loglevel: %s", logLevel)
		err = commandLine.Set("v", logLevel)
		if err != nil {
			glog.Warningf("Unable to set custom logLevel: %s", err.Error())
		}
	}
	err = commandLine.Parse(os.Args[1:])
	if err != nil {
		glog.Fatalf("Unable to parse command line options: %s", err.Error())
	}
}

func buildTeardownHelperFn(funcs ...func()) func() {
	return func() {
		for _, f := range funcs {
			if f != nil {
				f()
			}
		}
	}
}

// NewID creates a new unique ID used internally to CS
func (helper *Helper) NewID() string {
	return xid.New().String()
}

// RestURL ...
func (helper *Helper) RestURL(path string) string {
	var serverConfig *server.ServerConfig
	helper.Env.MustResolveAll(&serverConfig)

	protocol := "http"
	if serverConfig.EnableHTTPS {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s/api/rhacs/v1%s", protocol, serverConfig.BindAddress, path)
}

// NewRandAccount returns a random account that has the control plane team org id as its organisation id
// The org id value is taken from config/quota-management-list-configuration.yaml
func (helper *Helper) NewRandAccount() *amv1.Account {
	// this value is taken from config/quota-management-list-configuration.yaml
	orgID := "13640203"
	return helper.NewAccountWithNameAndOrg(faker.Name(), orgID)
}

// NewAccountWithNameAndOrg ...
func (helper *Helper) NewAccountWithNameAndOrg(name string, orgID string) *amv1.Account {
	account, err := helper.AuthHelper.NewAccount(helper.NewID(), name, faker.Email(), orgID)
	if err != nil {
		helper.T.Errorf("failed to create a new account: %s", err.Error())
	}
	return account
}

// NewAccount ...
func (helper *Helper) NewAccount(username, name, email string, orgID string) *amv1.Account {
	account, err := helper.AuthHelper.NewAccount(username, name, email, orgID)
	if err != nil {
		helper.T.Errorf(fmt.Sprintf("Unable to create a new account: %s", err.Error()))
	}
	return account
}

// NewAuthenticatedContext Returns an authenticated context that can be used with openapi functions
func (helper *Helper) NewAuthenticatedContext(account *amv1.Account, claims jwt.MapClaims) context.Context {
	token, err := helper.AuthHelper.CreateSignedJWT(account, claims)
	if err != nil {
		helper.T.Errorf(fmt.Sprintf("Unable to create a signed token: %s", err.Error()))
	}

	return context.WithValue(context.Background(), compat.ContextAccessToken, token)
}

// NewAuthenticatedAdminContext return an authenticated context that can be used with openapi function generated for the admin API
func (helper *Helper) NewAuthenticatedAdminContext(account *amv1.Account, claims jwt.MapClaims) context.Context {
	if claims == nil {
		claims = jwt.MapClaims{}
	}

	// do not override roles if explicitly defined
	if _, hasRealmAccess := claims["realm_access"]; !hasRealmAccess {
		claims["realm_access"] = map[string]interface{}{
			"roles": []string{"acs-fleet-manager-admin-full"},
		}
	}

	token, err := helper.AuthHelper.CreateSignedJWT(account, claims)
	if err != nil {
		helper.T.Errorf(fmt.Sprintf("Unable to create a signed token: %s", err.Error()))
	}

	return context.WithValue(context.Background(), adminprivate.ContextAccessToken, token)
}

// StartJWKCertServerMock ...
func (helper *Helper) StartJWKCertServerMock() (string, func()) {
	return mocks.NewJWKCertServerMock(helper.T, helper.JWTCA, auth.JwkKID)
}

// Migrations ...
func (helper *Helper) Migrations() (m []*db.Migration) {
	helper.Env.MustResolveAll(&m)
	return
}

// MigrateDB ...
func (helper *Helper) MigrateDB() {
	for _, migration := range helper.Migrations() {
		migration.Migrate()
	}
}

// CleanDB ...
func (helper *Helper) CleanDB() {
	for _, migration := range helper.Migrations() {
		migration.RollbackAll()
	}
}

// ResetDB ...
func (helper *Helper) ResetDB() {
	helper.CleanDB()
	helper.MigrateDB()
}

// CreateJWTString ...
func (helper *Helper) CreateJWTString(account *amv1.Account) string {
	token, err := helper.AuthHelper.CreateSignedJWT(account, nil)
	if err != nil {
		helper.T.Errorf(fmt.Sprintf("Unable to create a signed token: %s", err.Error()))
	}
	return token
}

// CreateJWTStringWithClaim ...
func (helper *Helper) CreateJWTStringWithClaim(account *amv1.Account, jwtClaims jwt.MapClaims) string {
	token, err := helper.AuthHelper.CreateSignedJWT(account, jwtClaims)
	if err != nil {
		helper.T.Errorf(fmt.Sprintf("Unable to create a signed token with the given claims: %s", err.Error()))
	}
	return token
}

// CreateDataPlaneJWTString creates a new JWT token for the dataplane (fleetshard) authorization
func (helper *Helper) CreateDataPlaneJWTString() string {
	claims := jwt.MapClaims{
		"iss": dataplaneIssuerURI,
		"aud": "acs-fleet-manager-private-api",
		"sub": "system:serviceaccount:rhacs:integration-tests",
	}
	token, err := helper.AuthHelper.CreateSignedJWT(nil, claims)
	if err != nil {
		helper.T.Errorf("Failed to create jwt token: %s", err.Error())
	}
	return token
}
