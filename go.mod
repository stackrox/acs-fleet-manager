module github.com/stackrox/acs-fleet-manager

go 1.16

require (
	github.com/Masterminds/squirrel v1.1.0
	github.com/Nerzal/gocloak/v8 v8.5.0
	github.com/antihax/optional v1.0.0
	github.com/auth0/go-jwt-middleware v0.0.0-20190805220309-36081240882b
	github.com/aws/aws-sdk-go v1.36.5
	github.com/blang/semver/v4 v4.0.0
	github.com/bxcodec/faker/v3 v3.2.0
	github.com/cenkalti/backoff/v4 v4.1.0 // indirect
	github.com/dgrijalva/jwt-go/v4 v4.0.0-preview1
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/go-healthcheck v0.1.0
	github.com/getsentry/sentry-go v0.3.1
	github.com/ghodss/yaml v1.0.0
	github.com/go-gormigrate/gormigrate/v2 v2.0.0
	github.com/go-resty/resty/v2 v2.3.0
	github.com/goava/di v1.11.0
	github.com/golang-jwt/jwt/v4 v4.2.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/google/uuid v1.2.0
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/mux v1.8.0
	github.com/lib/pq v1.10.4
	github.com/mattn/go-sqlite3 v1.14.3 // indirect
	github.com/mendsley/gojwk v0.0.0-20141217222730-4d5ec6e58103
	github.com/olekukonko/tablewriter v0.0.5
	github.com/onsi/gomega v1.17.0
	github.com/openshift-online/ocm-sdk-go v0.1.235
	github.com/openshift/api v3.9.0+incompatible
	github.com/operator-framework/api v0.3.25
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.0
	github.com/prometheus/client_golang v1.10.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.20.0
	github.com/rs/xid v1.2.1
	github.com/santhosh-tekuri/jsonschema/v3 v3.0.1
	github.com/selvatico/go-mocket v1.0.7
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/spyzhov/ajson v0.4.2
	github.com/xeipuuv/gojsonschema v1.2.0
	github.com/yaacov/tree-search-language v0.0.0-20190923184055-1c2dad2e354b
	github.com/zgalor/weberr v0.6.0
	golang.org/x/oauth2 v0.0.0-20210402161424-2e8d93401602
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/resty.v1 v1.12.0
	gopkg.in/yaml.v2 v2.4.0
	gorm.io/driver/postgres v1.0.8
	gorm.io/gorm v1.21.7
	k8s.io/api v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
)
