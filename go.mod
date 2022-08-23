module github.com/stackrox/acs-fleet-manager

go 1.17

require (
	github.com/Masterminds/squirrel v1.5.3
	github.com/Nerzal/gocloak/v11 v11.2.0
	github.com/antihax/optional v1.0.0
	github.com/auth0/go-jwt-middleware/v2 v2.0.1
	github.com/aws/aws-sdk-go v1.44.29
	github.com/blang/semver/v4 v4.0.0
	github.com/bxcodec/faker/v3 v3.8.0
	github.com/caarlos0/env/v6 v6.9.3
	github.com/docker/go-healthcheck v0.1.0
	github.com/getsentry/sentry-go v0.13.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-gormigrate/gormigrate/v2 v2.0.2
	github.com/go-resty/resty/v2 v2.7.0
	github.com/goava/di v1.11.1
	github.com/gogo/protobuf v1.3.2
	github.com/golang-jwt/jwt/v4 v4.4.1
	github.com/golang/glog v1.0.0
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.3.0
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/lib/pq v1.10.6
	github.com/mendsley/gojwk v0.0.0-20141217222730-4d5ec6e58103
	github.com/olekukonko/tablewriter v0.0.5
	github.com/onsi/ginkgo/v2 v2.1.4
	github.com/onsi/gomega v1.19.0
	github.com/openshift-online/ocm-sdk-go v0.1.286
	github.com/openshift/api v3.9.1-0.20191201231411-9f834e337466+incompatible
	github.com/operator-framework/api v0.4.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.0
	github.com/prometheus/client_golang v1.12.2
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.35.0
	github.com/redhat-developer/app-services-sdk-go/serviceaccounts v0.4.0
	github.com/rs/xid v1.4.0
	github.com/santhosh-tekuri/jsonschema/v3 v3.0.1
	github.com/selvatico/go-mocket v1.0.7
	github.com/spf13/cobra v1.4.0
	github.com/spf13/pflag v1.0.5
	github.com/spyzhov/ajson v0.7.1
	github.com/stackrox/rox v0.0.0-20210914215712-9ac265932e28
	github.com/stretchr/testify v1.7.2
	github.com/xeipuuv/gojsonschema v1.2.0
	github.com/yaacov/tree-search-language v0.0.0-20190923184055-1c2dad2e354b
	github.com/zgalor/weberr v0.6.0
	golang.org/x/oauth2 v0.0.0-20220608161450-d0670ef3b1eb
	golang.org/x/sys v0.0.0-20220610221304-9f5ed59c137d
	gopkg.in/resty.v1 v1.12.0
	gopkg.in/yaml.v2 v2.4.0
	gorm.io/driver/postgres v1.3.9
	gorm.io/gorm v1.23.8
	k8s.io/api v0.24.4
	k8s.io/apimachinery v0.24.4
	k8s.io/client-go v0.24.1
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9
	sigs.k8s.io/controller-runtime v0.12.1
)

require (
	github.com/antlr/antlr4/runtime/Go/antlr v0.0.0-20210826220005-b48c857c3a0e // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/cenkalti/backoff/v4 v4.1.3 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/distribution v2.8.1+incompatible // indirect
	github.com/emicklei/go-restful v2.15.0+incompatible // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/swag v0.21.1 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/gnostic v0.6.9 // indirect
	github.com/google/go-cmp v0.5.8 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/gorilla/css v1.0.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.12.1 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.3.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20200714003250-2b9c44734f2b // indirect
	github.com/jackc/pgtype v1.11.0 // indirect
	github.com/jackc/pgx/v4 v4.16.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mauricelam/genny v0.0.0-20190320071652-0800202903e5 // indirect
	github.com/microcosm-cc/bluemonday v1.0.18 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/segmentio/ksuid v1.0.4 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/stackrox/scanner v0.0.0-20220426001230-9ab6777c9581 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	go.uber.org/zap v1.21.0 // indirect
	golang.org/x/crypto v0.0.0-20220427172511-eb4f295cb31f // indirect
	golang.org/x/net v0.0.0-20220607020251-c690dde0001d // indirect
	golang.org/x/term v0.0.0-20220411215600-e5f449aeb171 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20220411224347-583f2d630306 // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220608133413-ed9918b62aac // indirect
	google.golang.org/grpc v1.47.0 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.24.0 // indirect
	k8s.io/component-base v0.24.0 // indirect
	k8s.io/klog/v2 v2.60.1 // indirect
	k8s.io/kube-openapi v0.0.0-20220413171646-5e7f5fdc6da6 // indirect
	sigs.k8s.io/json v0.0.0-20211208200746-9f7c6b3444d2 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.1 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace (
	github.com/antlr/antlr4 => github.com/antlr/antlr4/runtime/Go/antlr v0.0.0-20210930093333-01de314d7883
	github.com/facebookincubator/nvdtools => github.com/stackrox/nvdtools v0.0.0-20210326191554-5daeb6395b56
	github.com/fullsailor/pkcs7 => github.com/misberner/pkcs7 v0.0.0-20190417093538-a48bf0f78dea
	github.com/gogo/protobuf => github.com/connorgorman/protobuf v1.2.2-0.20210115205927-b892c1b298f7
	github.com/heroku/docker-registry-client => github.com/stackrox/docker-registry-client v0.0.0-20220204234128-07f109db0819
	github.com/operator-framework/helm-operator-plugins => github.com/stackrox/helm-operator v0.0.8-0.20220506091602-3764c49abfb3
	github.com/stackrox/rox => github.com/stackrox/stackrox v0.0.0-20220623061650-372b7a04e3ef
	go.uber.org/zap => github.com/stackrox/zap v1.15.1-0.20200720133746-810fd602fd0f
)

exclude k8s.io/client-go v12.0.0+incompatible
