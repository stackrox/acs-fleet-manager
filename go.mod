module github.com/stackrox/acs-fleet-manager

go 1.21

toolchain go1.21.9

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/antihax/optional v1.0.0
	github.com/aws/aws-sdk-go v1.51.15
	github.com/aws/aws-sdk-go-v2 v1.27.0
	github.com/aws/aws-sdk-go-v2/config v1.27.15
	github.com/aws/aws-sdk-go-v2/service/ses v1.22.8
	github.com/bxcodec/faker/v3 v3.8.1
	github.com/caarlos0/env/v6 v6.10.1
	github.com/containers/image/v5 v5.31.0
	github.com/coreos/go-oidc/v3 v3.10.0
	github.com/docker/go-healthcheck v0.1.0
	github.com/getsentry/sentry-go v0.27.0
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-gormigrate/gormigrate/v2 v2.1.2
	github.com/go-logr/logr v1.4.1
	github.com/goava/di v1.11.1
	github.com/gogo/protobuf v1.3.2
	github.com/golang-jwt/jwt/v4 v4.5.0
	github.com/golang/glog v1.2.0
	github.com/golang/protobuf v1.5.4
	github.com/google/go-cmp v0.6.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/gruntwork-io/terratest v0.46.14
	github.com/hashicorp/go-multierror v1.1.1
	github.com/lib/pq v1.10.9
	github.com/mendsley/gojwk v0.0.0-20141217222730-4d5ec6e58103
	github.com/olekukonko/tablewriter v0.0.5
	github.com/onsi/ginkgo/v2 v2.15.0
	github.com/onsi/gomega v1.31.1
	github.com/openshift-online/ocm-sdk-go v0.1.419
	github.com/openshift/addon-operator/apis v0.0.0-20231110045543-dd01f2f5c184
	github.com/openshift/api v0.0.0-20231117201702-2ea16bbab164
	github.com/operator-framework/api v0.22.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2
	github.com/prometheus/client_golang v1.19.0
	github.com/prometheus/client_model v0.6.0
	github.com/prometheus/common v0.53.0
	github.com/redhat-developer/app-services-sdk-core/app-services-sdk-go/serviceaccountmgmt v0.0.0-20230323122535-49460b57cc45
	github.com/rs/xid v1.5.0
	github.com/santhosh-tekuri/jsonschema/v3 v3.1.0
	github.com/selvatico/go-mocket v1.0.7
	github.com/spf13/cobra v1.8.0
	github.com/spf13/pflag v1.0.6-0.20210604193023-d5e0c0615ace
	github.com/spyzhov/ajson v0.9.1
	github.com/stackrox/rox v0.0.0-20230323083409-e83503a98fb4
	github.com/stretchr/testify v1.9.0
	github.com/xeipuuv/gojsonschema v1.2.0
	github.com/zgalor/weberr v0.8.2
	golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842
	golang.org/x/net v0.25.0
	golang.org/x/oauth2 v0.20.0
	golang.org/x/sync v0.7.0
	golang.org/x/sys v0.20.0
	gopkg.in/resty.v1 v1.12.0
	gopkg.in/yaml.v2 v2.4.0
	gorm.io/driver/postgres v1.5.7
	gorm.io/gorm v1.25.10
	helm.sh/helm/v3 v3.14.3
	k8s.io/api v0.29.2
	k8s.io/apimachinery v0.29.2
	k8s.io/autoscaler/vertical-pod-autoscaler v1.1.2
	k8s.io/client-go v0.29.2
	k8s.io/utils v0.0.0-20240102154912-e7106e64919e
	sigs.k8s.io/controller-runtime v0.17.2
	sigs.k8s.io/yaml v1.4.0
)

require (
	github.com/BurntSushi/toml v1.3.2 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/semver/v3 v3.2.1 // indirect
	github.com/Masterminds/sprig/v3 v3.2.3 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.15 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.7 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.7 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.20.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.24.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.28.9 // indirect
	github.com/aws/smithy-go v1.20.2 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/cloudflare/cfssl v1.6.4 // indirect
	github.com/containers/storage v1.54.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.3 // indirect
	github.com/cyphar/filepath-securejoin v0.2.5 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/evanphx/json-patch v5.7.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.8.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/go-jose/go-jose/v3 v3.0.3 // indirect
	github.com/go-jose/go-jose/v4 v4.0.1 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-sql-driver/mysql v1.7.1 // indirect
	github.com/go-task/slim-sprig v2.20.0+incompatible // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/gonvenience/bunt v1.3.5 // indirect
	github.com/gonvenience/neat v1.3.12 // indirect
	github.com/gonvenience/term v1.0.2 // indirect
	github.com/gonvenience/text v1.0.7 // indirect
	github.com/gonvenience/wrap v1.1.2 // indirect
	github.com/gonvenience/ytbx v1.4.4 // indirect
	github.com/google/certificate-transparency-go v1.1.7 // indirect
	github.com/google/gnostic-models v0.6.9-0.20230804172637-c7be7c783f49 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20231023181126-ff6d637d2a7b // indirect
	github.com/gorilla/css v1.0.0 // indirect
	github.com/gorilla/schema v1.2.1 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/gruntwork-io/go-commons v0.8.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/homeport/dyff v1.6.0 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20231201235250-de7065d80cb9 // indirect
	github.com/jackc/pgx/v5 v5.5.5 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jmoiron/sqlx v1.3.5 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-ciede2000 v0.0.0-20170301095244-782e8c62fec3 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/mattn/go-zglob v0.0.2-0.20190814121620-e3c945676326 // indirect
	github.com/microcosm-cc/bluemonday v1.0.23 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-ps v1.0.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/hashstructure v1.1.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/openshift/client-go v0.0.0-20230926161409-848405da69e1 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pquerna/otp v1.2.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring v0.66.0-rhobs1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/segmentio/analytics-go/v3 v3.3.0 // indirect
	github.com/segmentio/backo-go v1.0.1 // indirect
	github.com/sergi/go-diff v1.3.1 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/stackrox/scanner v0.0.0-20240110222630-351caa1e0024 // indirect
	github.com/texttheater/golang-levenshtein v1.0.1 // indirect
	github.com/tkuchiki/go-timezone v0.2.2 // indirect
	github.com/urfave/cli v1.22.14 // indirect
	github.com/virtuald/go-ordered-json v0.0.0-20170621173500-b18e6e673d74 // indirect
	github.com/weppos/publicsuffix-go v0.30.1-0.20230620154423-38c92ad2d5c6 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/zmap/zcrypto v0.0.0-20230310154051-c8b263fd8300 // indirect
	github.com/zmap/zlint/v3 v3.5.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/crypto v0.23.0 // indirect
	golang.org/x/exp/typeparams v0.0.0-20230307190834-24139beb5833 // indirect
	golang.org/x/term v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/tools v0.21.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	google.golang.org/genproto v0.0.0-20240123012728-ef4313101c80 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240123012728-ef4313101c80 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
	google.golang.org/grpc v1.62.1 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	honnef.co/go/tools v0.4.6 // indirect
	k8s.io/apiextensions-apiserver v0.29.2 // indirect
	k8s.io/component-base v0.29.2 // indirect
	k8s.io/klog/v2 v2.120.0 // indirect
	k8s.io/kube-openapi v0.0.0-20240105020646-a37d4de58910 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
)

replace (
	github.com/blevesearch/bleve => github.com/stackrox/bleve v0.0.0-20220907150529-4ecbd2543f9e
	github.com/facebookincubator/nvdtools => github.com/stackrox/nvdtools v0.0.0-20210326191554-5daeb6395b56
	github.com/fullsailor/pkcs7 => github.com/stackrox/pkcs7 v0.0.0-20220914154527-cfdb0aa47179
	github.com/gogo/protobuf => github.com/connorgorman/protobuf v1.2.2-0.20210115205927-b892c1b298f7
	github.com/heroku/docker-registry-client => github.com/stackrox/docker-registry-client v0.0.0-20230411213734-d75b95d65d28
	github.com/operator-framework/helm-operator-plugins => github.com/stackrox/helm-operator v0.0.12-0.20221003092512-fbf71229411f
	github.com/stackrox/rox => github.com/stackrox/stackrox v0.0.0-20240521145645-53dd371292a8 // v4.4.2
	go.uber.org/zap => github.com/stackrox/zap v1.15.1-0.20230918235618-2bd149903d0e
)

exclude k8s.io/client-go v12.0.0+incompatible
