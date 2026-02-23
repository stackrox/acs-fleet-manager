MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_PATH := $(patsubst %/,%,$(dir $(MKFILE_PATH)))
DOCS_DIR := $(PROJECT_PATH)/docs
TOOLS_DIR := $(PROJECT_PATH)/tools

.DEFAULT_GOAL := help
SHELL = bash

# The details of the application:
binary:=fleet-manager

# The image tag for building and pushing comes from TAG environment variable by default.
# Otherwise image tag is generated based on current commit hash.
# The version should be a 7-char hash from git. This is what the deployment process in app-interface expects.
ifeq ($(TAG),)
TAG=$(shell git rev-parse --short=7 HEAD)
endif
image_tag = $(TAG)

GINKGO_FLAGS ?= -v

# The version needs to be different for each deployment because otherwise the
# cluster will not pull the new image from the internal registry:
version:=$(shell date +%s)

NAMESPACE = rhacs
PROBE_NAMESPACE = rhacs-probe
IMAGE_NAME = fleet-manager
PROBE_IMAGE_NAME = probe
IMAGE_TARGET = standard
EMAILSENDER_IMAGE = emailsender

SHORT_IMAGE_REF = "$(IMAGE_NAME):$(image_tag)"
PROBE_SHORT_IMAGE_REF = "$(PROBE_IMAGE_NAME):$(image_tag)"
EMAILSENDER_SHORT_IMAGE_REF = "$(EMAILSENDER_IMAGE):$(image_tag)"

image_repository:=$(IMAGE_NAME)
probe_image_repository:=$(PROBE_IMAGE_NAME)
emailsender_image_repository:=$(EMAILSENDER_IMAGE)

# In the development environment we are pushing the image directly to the image
# registry inside the development cluster. That registry has a different name
# when it is accessed from outside the cluster and when it is accessed from
# inside the cluster. We need the external name to push the image, and the
# internal name to pull it.
external_image_registry:=quay.io/rhacs-eng
internal_image_registry:=image-registry.openshift-image-registry.svc:5000

DOCKER ?= docker

# Default Variables
ENABLE_OCM_MOCK ?= true
OCM_MOCK_MODE ?= emulate-server
GITOPS_CONFIG_FILE ?= ${PROJECT_PATH}/dev/config/gitops-config.yaml
DATAPLANE_CLUSTER_CONFIG_FILE ?= ${PROJECT_PATH}/dev/config/dataplane-cluster-configuration.yaml
PROVIDERS_CONFIG_FILE ?= ${PROJECT_PATH}/dev/config/provider-configuration.yaml
QUOTA_MANAGEMENT_LIST_CONFIG_FILE ?= ${PROJECT_PATH}/dev/config/quota-management-list-configuration.yaml

GO := go
GOFMT := gofmt
# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set).
ifeq (,$(shell $(GO) env GOBIN))
GOBIN=$(shell $(GO) env GOPATH)/bin
else
GOBIN=$(shell $(GO) env GOBIN)
endif

ifeq ($(IMAGE_PLATFORM),)
IMAGE_PLATFORM=linux/$(shell $(GO) env GOARCH)
endif

ifeq ($(CLUSTER_DNS),)
# This makes sure that the "ingresscontroller" kind, which only exists on OpenShift by default, is only queried
# when CLUSTER_DNS is not set.
CLUSTER_DNS=$(shell oc get -n "openshift-ingress-operator" ingresscontrollers default -o=jsonpath='{.status.domain}' --ignore-not-found 2> /dev/null)
ifeq ($(CLUSTER_DNS),)
CLUSTER_DNS=host.acscs.internal
endif
endif

LOCAL_BIN_PATH := ${PROJECT_PATH}/bin
# Add the project-level bin directory into PATH. Needed in order
# for `go generate` to use project-level bin directory binaries first
export PATH := ${LOCAL_BIN_PATH}:$(PATH)

GOTESTSUM_BIN := $(LOCAL_BIN_PATH)/gotestsum
$(GOTESTSUM_BIN): $(TOOLS_DIR)/go.mod $(TOOLS_DIR)/go.sum
	@cd $(TOOLS_DIR) && GOBIN=${LOCAL_BIN_PATH} $(GO) install gotest.tools/gotestsum

MOQ_BIN := $(LOCAL_BIN_PATH)/moq
$(MOQ_BIN): $(TOOLS_DIR)/go.mod $(TOOLS_DIR)/go.sum
	@cd $(TOOLS_DIR) && GOBIN=${LOCAL_BIN_PATH} $(GO) install github.com/matryer/moq

CHAMBER_BIN := $(LOCAL_BIN_PATH)/chamber
$(CHAMBER_BIN): $(TOOLS_DIR)/go.mod $(TOOLS_DIR)/go.sum
	@cd $(TOOLS_DIR) && GOBIN=${LOCAL_BIN_PATH} $(GO) install github.com/segmentio/chamber/v2

GINKGO_BIN := $(LOCAL_BIN_PATH)/ginkgo
$(GINKGO_BIN): go.mod go.sum
	@GOBIN=${LOCAL_BIN_PATH} $(GO) install github.com/onsi/ginkgo/v2/ginkgo

TOOLS_VENV_DIR := $(LOCAL_BIN_PATH)/tools_venv
$(TOOLS_VENV_DIR): $(TOOLS_DIR)/requirements.txt
	@set -e; \
	trap "rm -rf $(TOOLS_VENV_DIR)" ERR; \
	python3 -m venv $(TOOLS_VENV_DIR); \
	. $(TOOLS_VENV_DIR)/bin/activate; \
	pip install --upgrade pip==23.3.1; \
	pip install -r $(TOOLS_DIR)/requirements.txt; \
	touch $(TOOLS_VENV_DIR) # update directory modification timestamp even if no changes were made by pip. This will allow to skip this target if the directory is up-to-date

OPENAPI_GENERATOR ?= ${LOCAL_BIN_PATH}/openapi-generator
NPM ?= "$(shell which npm 2> /dev/null)"
openapi-generator:
ifeq (, $(shell which ${NPM} 2> /dev/null))
	@echo "npm is not available please install it to be able to install openapi-generator"
	exit 1
endif
ifeq (, $(shell which ${LOCAL_BIN_PATH}/openapi-generator 2> /dev/null))
	@{ \
	set -e ;\
	mkdir -p ${LOCAL_BIN_PATH} ;\
	mkdir -p ${LOCAL_BIN_PATH}/openapi-generator-installation ;\
	cd ${LOCAL_BIN_PATH} ;\
	${NPM} install --prefix ${LOCAL_BIN_PATH}/openapi-generator-installation @openapitools/openapi-generator-cli@cli-4.3.1 ;\
	ln -s openapi-generator-installation/node_modules/.bin/openapi-generator openapi-generator ;\
	}
endif

SPECTRAL ?= ${LOCAL_BIN_PATH}/spectral
NPM ?= "$(shell which npm 2> /dev/null)"
specinstall:
ifeq (, $(shell which ${NPM} 2> /dev/null))
	@echo "npm is not available please install it to be able to install spectral"
	exit 1
endif
ifeq (, $(shell which ${LOCAL_BIN_PATH}/spectral 2> /dev/null))
	@{ \
	set -e ;\
	mkdir -p ${LOCAL_BIN_PATH} ;\
	mkdir -p ${LOCAL_BIN_PATH}/spectral-installation ;\
	cd ${LOCAL_BIN_PATH} ;\
	${NPM} install --prefix ${LOCAL_BIN_PATH}/spectral-installation @stoplight/spectral-cli ;\
	${NPM} i --prefix ${LOCAL_BIN_PATH}/spectral-installation @rhoas/spectral-ruleset ;\
	ln -s spectral-installation/node_modules/.bin/spectral spectral ;\
	}
endif
openapi/spec/validate: specinstall
	spectral lint openapi/fleet-manager.yaml openapi/fleet-manager-private-admin.yaml


ifeq ($(shell uname -s | tr A-Z a-z), darwin)
        PGHOST:="127.0.0.1"
else
        PGHOST:="172.18.0.22"
endif

ifeq ($(shell echo ${DEBUG}), 1)
	GOARGS := $(GOARGS) -gcflags=all="-N -l"
endif

### Environment-sourced variables with defaults
# Can be overriden by setting environment var before running
# Example:
#   OCM_ENV=testing make run
#   export OCM_ENV=testing; make run
# Set the environment to development by default
ifndef OCM_ENV
	OCM_ENV:=integration
endif

GOTESTSUM_FORMAT ?= standard-verbose

# Enable Go modules:
export GO111MODULE=on
export GOPROXY=https://proxy.golang.org
export GOPRIVATE=gitlab.cee.redhat.com

ifndef SERVER_URL
	SERVER_URL:=http://localhost:8000
endif

ifndef TEST_TIMEOUT
	ifeq ($(OCM_ENV), integration)
		TEST_TIMEOUT=30m
	else
		TEST_TIMEOUT=5h
	endif
endif

# Prints a list of useful targets.
help:
	@echo "Central Service Fleet Manager make targets"
	@echo ""
	@echo "make verify                      verify source code"
	@echo "make lint                        lint go files and .yaml templates"
	@echo "make binary                      compile binaries"
	@echo "make install                     compile binaries and install in GOPATH bin"
	@echo "make run                         run the application"
	@echo "make run/docs                    run swagger and host the api spec"
	@echo "make test                        run unit tests"
	@echo "make test/integration            run integration tests"
	@echo "make code/check                  fail if formatting is required"
	@echo "make code/fix                    format files"
	@echo "make generate                    generate go and openapi modules"
	@echo "make openapi/generate            generate openapi modules"
	@echo "make openapi/validate            validate openapi schema"
	@echo "make image/build                 build fleet-manager and fleetshard-sync container image"
	@echo "make image/push                  push image"
	@echo "make setup/git/hooks             setup git hooks"
	@echo "make secrets/touch               touch all required secret files"
	@echo "make docker/login/internal       login to an openshift cluster image registry"
	@echo "make image/push/internal         push image to an openshift cluster image registry."
	@echo "make deploy/project              deploy the service via templates to an openshift cluster"
	@echo "make undeploy                    remove the service deployments from an openshift cluster"
	@echo "make redhatsso/setup             setup sso clientId & clientSecret"
	@echo "make centralidp/setup            setup Central's static auth config (client_secret)"
	@echo "make openapi/spec/validate       validate OpenAPI spec using spectral"
	@echo "$(fake)"
.PHONY: help

all: openapi/generate binary
.PHONY: all

# Install pre-commit hooks
.PHONY: setup/git/hooks
setup/git/hooks:
	-git config --unset-all core.hooksPath
	@if command -v pre-commit >/dev/null 2>&1; then \
		echo "Installing pre-commit hooks"; \
		pre-commit install; \
	else \
		echo "Please install pre-commit: See https://pre-commit.com/index.html for installation instructions."; \
		echo "Re-run 'make setup/git/hooks' setup step after pre-commit has been installed."; \
	fi

# Checks if a GOPATH is set, or emits an error message
check-gopath:
ifndef GOPATH
	$(error GOPATH is not set)
endif
.PHONY: check-gopath

# Verifies that source passes standard checks.
# Also verifies that the OpenAPI spec is correct.
verify: check-gopath openapi/validate
	$(GO) vet \
		./cmd/... \
		./pkg/... \
		./internal/... \
		./test/... \
		./fleetshard/... \
		./probe/... \
		./emailsender/... \
		./deploy/test/...
.PHONY: verify

# Runs linter against go files and .y(a)ml files in the templates directory
# Requires pre-commit to be installed: See https://pre-commit.com/index.html for installation instructions.
# and spectral installed via npm
lint: specinstall
	pre-commit run golangci-lint --all-files
	spectral lint templates/*.yml templates/*.yaml --ignore-unknown-format --ruleset .validate-templates.yaml
.PHONY: lint

pre-commit:
	pre-commit run --files $(git --no-pager diff --name-only main)
.PHONY: pre-commit

# Build binaries
# NOTE it may be necessary to use CGO_ENABLED=0 for backwards compatibility with centos7 if not using centos7

fleet-manager:
	GOOS="$(GOOS)" GOARCH="$(GOARCH)" CGO_ENABLED=0 $(GO) build $(GOARGS) ./cmd/fleet-manager
.PHONY: fleet-manager

fleetshard-sync:
	GOOS="$(GOOS)" GOARCH="$(GOARCH)" CGO_ENABLED=0  $(GO) build $(GOARGS) -o fleetshard-sync ./fleetshard
.PHONY: fleetshard-sync

probe:
	GOOS="$(GOOS)" GOARCH="$(GOARCH)" CGO_ENABLED=0 $(GO) build $(GOARGS) -o probe/bin/probe ./probe/cmd/probe
.PHONY: probe

acsfleetctl:
	GOOS="$(GOOS)" GOARCH="$(GOARCH)" CGO_ENABLED=0  $(GO) build $(GOARGS) -o acsfleetctl ./cmd/acsfleetctl
.PHONY: acsfleetctl

emailsender:
	GOOS="$(GOOS)" GOARCH="$(GOARCH)" CGO_ENABLED=0  $(GO) build $(GOARGS) -o emailsender/bin/emailsender ./emailsender/cmd/app
.PHONY: emailsender

binary: fleet-manager fleetshard-sync probe acsfleetctl emailsender
.PHONY: binary

clean:
	rm -f fleet-manager fleetshard-sync probe/bin/probe emailsender/bin/emailsender
.PHONY: clean

# Runs the unit tests.
#
# Args:
#   TESTFLAGS: Flags to pass to `go test`. The `-v` argument is always passed.
#
# Examples:
#   make test TESTFLAGS="-run TestSomething"
test: $(GOTESTSUM_BIN)
	OCM_ENV=testing $(GOTESTSUM_BIN) --junitfile data/results/unit-tests.xml --format $(GOTESTSUM_FORMAT) -- -p 1 -v -count=1 $(TESTFLAGS) \
		$(shell go list ./... | grep -v /test)
.PHONY: test

# Runs the AWS integration tests.
test/aws: $(GOTESTSUM_BIN)
	RUN_AWS_INTEGRATION=true \
	$(GOTESTSUM_BIN) --junitfile data/results/aws-integration-tests.xml --format $(GOTESTSUM_FORMAT) -- -p 1 -v -timeout 45m -count=1 \
		./fleetshard/pkg/central/cloudprovider/awsclient/... \
		./fleetshard/pkg/cipher/... \
		./emailsender/pkg/email/...
.PHONY: test/aws

# Runs the integration tests.
#
# Args:
#   TESTFLAGS: Flags to pass to `go test`. The `-v` argument is always passed.
#
# Example:
#   make test/integration
#   make test/integration TESTFLAGS="-run TestAccounts"     acts as TestAccounts* and run TestAccountsGet, TestAccountsPost, etc.
#   make test/integration TESTFLAGS="-run TestAccountsGet"  runs TestAccountsGet
#   make test/integration TESTFLAGS="-short"                skips long-run tests
test/integration/central: $(GOTESTSUM_BIN)
	OCM_ENV=integration $(GOTESTSUM_BIN) --junitfile data/results/fleet-manager-integration-tests.xml --format $(GOTESTSUM_FORMAT) -- -p 1 -ldflags -s -v -timeout $(TEST_TIMEOUT) -count=1 $(TESTFLAGS) \
				./internal/central/test/integration/...
.PHONY: test/integration/central

test/deploy: $(GOTESTSUM_BIN)
	$(GOTESTSUM_BIN) --format $(GOTESTSUM_FORMAT) -- -p 1 -ldflags -s -v -timeout $(TEST_TIMEOUT) -count=1 $(TESTFLAGS) \
				./deploy/test...
.PHONY: test/deploy

test/integration: test/integration/central test/deploy
.PHONY: test/integration

# remove OSD cluster after running tests against real OCM
# requires OCM_OFFLINE_TOKEN env var exported
test/cluster/cleanup:
	./scripts/cleanup_test_cluster.sh
.PHONY: test/cluster/cleanup

# Runs E2E test suite
#
# Examples:
#    make test/e2e
#    make test/e2e GINKGO_FLAGS="-v --focus='should be created and deployed to k8s'" -- runs a specific test
test/e2e: $(GINKGO_BIN)
	CLUSTER_ID=1234567890abcdef1234567890abcdef \
	RUN_E2E=true \
	ENABLE_CENTRAL_EXTERNAL_DOMAIN=$(ENABLE_CENTRAL_EXTERNAL_DOMAIN) \
	GITOPS_CONFIG_PATH=$(GITOPS_CONFIG_FILE) \
	$(GINKGO_BIN) -r $(GINKGO_FLAGS) \
		--randomize-suites \
		--fail-on-pending --keep-going \
		--cover --coverprofile=cover.profile \
		--race --trace \
		--json-report=e2e-report.json \
		--timeout=$(TEST_TIMEOUT) \
		--poll-progress-after=5m \
		 ./e2e/...
.PHONY: test/e2e

test/e2e/multicluster: $(GINKGO_BIN)
	CLUSTER_ID=1234567890abcdef1234567890abcdef \
	ENABLE_CENTRAL_EXTERNAL_DOMAIN=$(ENABLE_CENTRAL_EXTERNAL_DOMAIN) \
	GITOPS_CONFIG_PATH=$(GITOPS_CONFIG_FILE) \
	RUN_MULTICLUSTER_E2E=true \
	$(GINKGO_BIN) -r $(GINKGO_FLAGS) \
		--randomize-suites \
		--fail-on-pending --keep-going \
		--cover --coverprofile=cover.profile \
		--race --trace \
		--json-report=e2e-report.json \
		--timeout=$(TEST_TIMEOUT) \
		--poll-progress-after=5m \
		 ./e2e/multicluster/...
.PHONY: test/e2e/multicluster

# Deploys the necessary applications to the selected cluster and runs e2e tests inside the container
# Useful for debugging Openshift CI runs locally
test/deploy/e2e-dockerized:
	./.openshift-ci/e2e-runtime/e2e_dockerized.sh
.PHONY: test/deploy/e2e-dockerized

test/e2e/reset:
	@./dev/env/scripts/reset
.PHONY: test/e2e/reset

test/e2e/cleanup:
	@./dev/env/scripts/down.sh
.PHONY: test/e2e/cleanup

# generate files
generate: $(MOQ_BIN) openapi/generate
	$(GO) generate ./...
.PHONY: generate

# validate the openapi schema
openapi/validate: openapi-generator
	$(OPENAPI_GENERATOR) validate -i openapi/fleet-manager.yaml
	$(OPENAPI_GENERATOR) validate -i openapi/fleet-manager-private.yaml
	$(OPENAPI_GENERATOR) validate -i openapi/fleet-manager-private-admin.yaml
	$(OPENAPI_GENERATOR) validate -i openapi/emailsender.yaml
.PHONY: openapi/validate

# generate the openapi schema and generated package
openapi/generate: openapi/generate/public openapi/generate/private openapi/generate/admin openapi/generate/rhsso openapi/generate/emailsender
.PHONY: openapi/generate

openapi/generate/public: openapi-generator
	rm -rf internal/central/pkg/api/public
	$(OPENAPI_GENERATOR) validate -i openapi/fleet-manager.yaml
	$(OPENAPI_GENERATOR) generate -i openapi/fleet-manager.yaml -g go -o internal/central/pkg/api/public --package-name public -t openapi/templates --ignore-file-override ./.openapi-generator-ignore
	$(GOFMT) -w internal/central/pkg/api/public
.PHONY: openapi/generate/public

openapi/generate/private: openapi-generator
	rm -rf internal/central/pkg/api/private
	$(OPENAPI_GENERATOR) validate -i openapi/fleet-manager-private.yaml
	$(OPENAPI_GENERATOR) generate -i openapi/fleet-manager-private.yaml -g go -o internal/central/pkg/api/private --package-name private -t openapi/templates --ignore-file-override ./.openapi-generator-ignore
	$(GOFMT) -w internal/central/pkg/api/private
.PHONY: openapi/generate/private

openapi/generate/admin: openapi-generator
	rm -rf internal/central/pkg/api/admin/private
	$(OPENAPI_GENERATOR) validate -i openapi/fleet-manager-private-admin.yaml
	$(OPENAPI_GENERATOR) generate -i openapi/fleet-manager-private-admin.yaml -g go -o internal/central/pkg/api/admin/private --package-name private -t openapi/templates --ignore-file-override ./.openapi-generator-ignore
	$(GOFMT) -w internal/central/pkg/api/admin/private
.PHONY: openapi/generate/admin

openapi/generate/rhsso: openapi-generator
	rm -rf pkg/client/redhatsso/api
	$(OPENAPI_GENERATOR) validate -i openapi/rh-sso-dynamic-client.yaml
	$(OPENAPI_GENERATOR) generate -i openapi/rh-sso-dynamic-client.yaml -g go -o pkg/client/redhatsso/api --package-name api -t openapi/templates --ignore-file-override ./.openapi-generator-ignore
	$(GOFMT) -w pkg/client/redhatsso/api
.PHONY: openapi/generate/rhsso

openapi/generate/emailsender: openapi-generator
	rm -rf emailsender/pkg/client/openapi
	$(OPENAPI_GENERATOR) validate -i openapi/emailsender.yaml
	$(OPENAPI_GENERATOR) generate -i openapi/emailsender.yaml -g go -o emailsender/pkg/client/openapi --package-name openapi -t openapi/templates --ignore-file-override ./.openapi-generator-ignore
	$(GOFMT) -w emailsender/pkg/client/openapi
.PHONY: openapi/generate/emailsender

# fail if formatting is required
code/check:
	@if ! [ -z "$$(find . -path './vendor' -prune -o -type f -name '*.go' -print0 | xargs -0 $(GOFMT) -l)" ]; then \
		echo "Please run 'make code/fix'."; \
		false; \
	fi
.PHONY: code/check

# clean up code and dependencies
code/fix:
	@$(GO) mod tidy
	@$(GOFMT) -w `find . -type f -name '*.go' -not -path "./vendor/*"`
.PHONY: code/fix

run: fleet-manager db/migrate
	./fleet-manager serve \
		--dataplane-cluster-config-file $(DATAPLANE_CLUSTER_CONFIG_FILE)
.PHONY: run

# Run Swagger and host the api docs
run/docs:
	$(DOCKER) run -u $(shell id -u) --rm --name swagger_ui_docs -d -p 8082:8080 -e URLS="[ \
		{ url: \"./openapi/fleet-manager.yaml\", name: \"Public API\" },\
		{ url: \"./openapi/fleet-manager-private.yaml\", name: \"Private API\"},\
		{ url: \"./openapi/fleet-manager-private-admin.yaml\", name: \"Private Admin API\"},\
		{ url: \"./openapi/emailsender.yaml\", name: \"Emailsender API\"}]"\
		  -v $(PWD)/openapi/:/usr/share/nginx/html/openapi:Z swaggerapi/swagger-ui
	@echo "Please open http://localhost:8082/"
.PHONY: run/docs

# Remove Swagger container
run/docs/teardown:
	$(DOCKER) container stop swagger_ui_docs
	$(DOCKER) container rm swagger_ui_docs
.PHONY: run/docs/teardown

db/setup:
	./scripts/local_db_setup.sh
.PHONY: db/setup

db/start:
	$(DOCKER) start fleet-manager-db
.PHONY: db/start

db/migrate:
	$(GO) run ./cmd/fleet-manager migrate
.PHONY: db/migrate

db/teardown:
	./scripts/local_db_teardown.sh
.PHONY: db/teardown

db/login:
	$(DOCKER) exec -u $(shell id -u) -it fleet-manager-db /bin/bash -c "PGPASSWORD=$(shell cat secrets/db.password) psql -d $(shell cat secrets/db.name) -U $(shell cat secrets/db.user)"
.PHONY: db/login

db/psql:
	@PGPASSWORD=$(shell cat secrets/db.password) psql -h localhost -d $(shell cat secrets/db.name) -U $(shell cat secrets/db.user)
.PHONY: db/psql

db/generate/insert/cluster:
	@read -r id external_id provider region multi_az<<<"$(shell ocm get /api/clusters_mgmt/v1/clusters/${CLUSTER_ID} | jq '.id, .external_id, .cloud_provider.id, .region.id, .multi_az' | tr -d \" | xargs -n2 echo)";\
	echo -e "Run this command in your database:\n\nINSERT INTO clusters (id, created_at, updated_at, cloud_provider, cluster_id, external_id, multi_az, region, status, provider_type) VALUES ('"$$id"', current_timestamp, current_timestamp, '"$$provider"', '"$$id"', '"$$external_id"', "$$multi_az", '"$$region"', 'cluster_provisioned', 'ocm');";
.PHONY: db/generate/insert/cluster

# Login to the OpenShift internal registry
docker/login/internal:
	@$(DOCKER) login -u kubeadmin --password-stdin <<< $(shell oc whoami -t) $(shell oc get route default-route -n openshift-image-registry -o jsonpath="{.spec.host}")
.PHONY: docker/login/internal

# Login to registry.redhat.io
docker/login/rh-registry:
	@$(DOCKER) login -u "${RH_REGISTRY_USER}" --password-stdin <<< "${RH_REGISTRY_PW}" registry.redhat.io
.PHONY: docker/login/rh-registry

# Build the image
image/build:
	$(DOCKER) buildx build -t $(SHORT_IMAGE_REF) . --load
	@echo "New image tag: $(SHORT_IMAGE_REF). You might want to"
	@echo "export FLEET_MANAGER_IMAGE=$(SHORT_IMAGE_REF)"
ifeq ("$(CLUSTER_TYPE)","kind")
	@echo "Loading image into kind"
	kind load docker-image $(SHORT_IMAGE_REF)
endif
.PHONY: image/build

image/build/probe: GOOS=linux
image/build/probe: IMAGE_REF="$(external_image_registry)/$(probe_image_repository):$(image_tag)"
image/build/probe:
	$(DOCKER) build -t $(IMAGE_REF) -f probe/Dockerfile .
	$(DOCKER) tag $(IMAGE_REF) $(PROBE_SHORT_IMAGE_REF)
.PHONY: image/build/probe

image/build/emailsender: GOOS=linux
image/build/emailsender: IMAGE_REF="$(external_image_registry)/$(emailsender_image_repository):$(image_tag)"
image/build/emailsender:
	$(DOCKER) build -t $(IMAGE_REF) -f emailsender/Dockerfile .
	$(DOCKER) tag $(IMAGE_REF) $(EMAILSENDER_SHORT_IMAGE_REF)
.PHONY: image/build/emailsender

image/push/emailsender: IMAGE_REF="$(external_image_registry)/$(emailsender_image_repository):$(image_tag)"
image/push/emailsender: image/build/emailsender
	$(DOCKER) push $(IMAGE_REF)
	@echo
	@echo "emailsender image was pushed as $(IMAGE_REF)."
.PHONY: image/push/emailsender

# Build and push the image
image/push: image/push/fleet-manager image/push/probe
.PHONY: image/push

image/push/fleet-manager: IMAGE_REF="$(external_image_registry)/$(image_repository):$(image_tag)"
image/push/fleet-manager:
	$(DOCKER) buildx build -t $(IMAGE_REF) --platform $(IMAGE_PLATFORM) --push .
	@echo
	@echo "Image was pushed as $(IMAGE_REF). You might want to"
	@echo "export FLEET_MANAGER_IMAGE=$(IMAGE_REF)"
.PHONY: image/push/fleet-manager

image/push/probe: IMAGE_REF="$(external_image_registry)/$(probe_image_repository):$(image_tag)"
image/push/probe: image/build/probe
	$(DOCKER) push $(IMAGE_REF)
	@echo
	@echo "Image was pushed as $(IMAGE_REF)."
.PHONY: image/push/probe

# push the image to the OpenShift internal registry
image/push/internal: IMAGE_TAG ?= $(image_tag)
image/push/internal: docker/login/internal
	@oc get imagestream $(IMAGE_NAME) -n $(NAMESPACE) >/dev/null 2>&1 || oc create imagestream $(IMAGE_NAME) -n $(NAMESPACE) --lookup-local
	$(DOCKER) buildx build -t "$(shell oc get route default-route -n openshift-image-registry -o jsonpath="{.spec.host}")/$(NAMESPACE)/$(IMAGE_NAME):$(IMAGE_TAG)" --platform linux/amd64 --push .
.PHONY: image/push/internal

# Touch all necessary secret files for fleet manager to start up
secrets/touch:
	touch secrets/aws.accesskey \
          secrets/aws.accountid \
          secrets/aws.route53accesskey \
          secrets/aws.route53secretaccesskey \
          secrets/aws.secretaccesskey \
          secrets/db.host \
          secrets/db.name \
          secrets/db.password \
          secrets/db.port \
          secrets/db.user \
          secrets/central.idp-client-secret \
          secrets/ocm-service.clientId \
          secrets/ocm-service.clientSecret \
          secrets/ocm-service.token \
          secrets/rhsso-logs.clientId \
          secrets/rhsso-logs.clientSecret \
          secrets/rhsso-metrics.clientId \
          secrets/rhsso-metrics.clientSecret \
          secrets/redhatsso-service.clientId \
          secrets/redhatsso-service.clientSecret
.PHONY: secrets/touch

# Setup for AWS credentials
aws/setup:
	@echo -n "$(AWS_ACCOUNT_ID)" > secrets/aws.accountid
	@echo -n "$(AWS_ACCESS_KEY)" > secrets/aws.accesskey
	@echo -n "$(AWS_SECRET_ACCESS_KEY)" > secrets/aws.secretaccesskey
	@echo -n "$(ROUTE53_ACCESS_KEY)" > secrets/aws.route53accesskey
	@echo -n "$(ROUTE53_SECRET_ACCESS_KEY)" > secrets/aws.route53secretaccesskey
.PHONY: aws/setup

redhatsso/setup:
	@echo -n "$(SSO_CLIENT_ID)" > secrets/redhatsso-service.clientId
	@echo -n "$(SSO_CLIENT_SECRET)" > secrets/redhatsso-service.clientSecret
.PHONY:redhatsso/setup

# Setup for the Central's IdP integration
centralidp/setup:
	@echo -n "$(CENTRAL_IDP_CLIENT_SECRET)" > secrets/central.idp-client-secret
.PHONY:centralidp/setup

# Setup dummy OCM_OFFLINE_TOKEN for integration testing
ocm/setup: OCM_OFFLINE_TOKEN ?= "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c" # pragma: allowlist secret
ocm/setup:
	@echo -n "$(OCM_OFFLINE_TOKEN)" > secrets/ocm-service.token
	@echo -n "" > secrets/ocm-service.clientId
	@echo -n "" > secrets/ocm-service.clientSecret
.PHONY: ocm/setup

# create project where the service will be deployed in an OpenShift cluster
deploy/project:
	@-oc new-project $(NAMESPACE)
.PHONY: deploy/project

# deploy the postgres database required by the service to an OpenShift cluster
deploy/db:
	oc process -f ./templates/db-template.yml --local | oc apply -f - -n $(NAMESPACE)
	@time timeout --foreground 3m bash -c "until oc get pods -n $(NAMESPACE) | grep fleet-manager-db | grep -v deploy | grep -q Running; do echo 'database is not ready yet'; sleep 10; done"
.PHONY: deploy/db

# deploys the secrets required by the service to an OpenShift cluster
deploy/secrets:
	@oc process -f ./templates/secrets-template.yml --local \
		-p DATABASE_HOST="fleet-manager-db" \
		-p OCM_SERVICE_CLIENT_ID="$(shell ([ -s './secrets/ocm-service.clientId' ] && [ -z '${OCM_SERVICE_CLIENT_ID}' ]) && cat ./secrets/ocm-service.clientId || echo '${OCM_SERVICE_CLIENT_ID}')" \
		-p OCM_SERVICE_CLIENT_SECRET="$(shell ([ -s './secrets/ocm-service.clientSecret' ] && [ -z '${OCM_SERVICE_CLIENT_SECRET}' ]) && cat ./secrets/ocm-service.clientSecret || echo '${OCM_SERVICE_CLIENT_SECRET}')" \
		-p OCM_SERVICE_TOKEN="$(shell ([ -s './secrets/ocm-service.token' ] && [ -z '${OCM_SERVICE_TOKEN}' ]) && cat ./secrets/ocm-service.token || echo '${OCM_SERVICE_TOKEN}')" \
		-p AWS_ACCESS_KEY="$(shell ([ -s './secrets/aws.accesskey' ] && [ -z '${AWS_ACCESS_KEY}' ]) && cat ./secrets/aws.accesskey || echo '${AWS_ACCESS_KEY}')" \
		-p AWS_ACCOUNT_ID="$(shell ([ -s './secrets/aws.accountid' ] && [ -z '${AWS_ACCOUNT_ID}' ]) && cat ./secrets/aws.accountid || echo '${AWS_ACCOUNT_ID}')" \
		-p AWS_SECRET_ACCESS_KEY="$(shell ([ -s './secrets/aws.secretaccesskey' ] && [ -z '${AWS_SECRET_ACCESS_KEY}' ]) && cat ./secrets/aws.secretaccesskey || echo '${AWS_SECRET_ACCESS_KEY}')" \
		-p ROUTE53_ACCESS_KEY="$(shell ([ -s './secrets/aws.route53accesskey' ] && [ -z '${ROUTE53_ACCESS_KEY}' ]) && cat ./secrets/aws.route53accesskey || echo '${ROUTE53_ACCESS_KEY}')" \
		-p ROUTE53_SECRET_ACCESS_KEY="$(shell ([ -s './secrets/aws.route53secretaccesskey' ] && [ -z '${ROUTE53_SECRET_ACCESS_KEY}' ]) && cat ./secrets/aws.route53secretaccesskey || echo '${ROUTE53_SECRET_ACCESS_KEY}')" \
		-p SSO_CLIENT_ID="$(shell ([ -s './secrets/redhatsso-service.clientId' ] && [ -z '${SSO_CLIENT_ID}' ]) && cat ./secrets/redhatsso-service.clientId || echo '${SSO_CLIENT_ID}')" \
		-p SSO_CLIENT_SECRET="$(shell ([ -s './secrets/redhatsso-service.clientSecret' ] && [ -z '${SSO_CLIENT_SECRET}' ]) && cat ./secrets/redhatsso-service.clientSecret || echo '${SSO_CLIENT_SECRET}')" \
		-p CENTRAL_IDP_CLIENT_SECRET="$(shell ([ -s './secrets/central.idp-client-secret' ] && [ -z '${CENTRAL_IDP_CLIENT_SECRET}' ]) && cat ./secrets/central.idp-client-secret || echo '${CENTRAL_IDP_CLIENT_SECRET}')" \
		| oc apply -f - -n $(NAMESPACE)
.PHONY: deploy/secrets

deploy/envoy:
	@oc apply -f ./templates/envoy-config-configmap.yml -n $(NAMESPACE)
.PHONY: deploy/envoy

deploy/route:
	@oc process -f ./templates/route-template.yml --local | oc apply -f - -n $(NAMESPACE)
.PHONY: deploy/route

# When making changes to the gitops configuration for development purposes
# situated here dev/env/manifests/fleet-manager/04-gitops-config.yaml, this
# target will update the gitops configmap on the dev cluster.
# It might take a few seconds/minutes for fleet-manager to observe the changes.
# Changes to the configmap are hot-reloaded
# See https://kubernetes.io/docs/concepts/configuration/configmap/#mounted-configmaps-are-updated-automatically
deploy/gitops:
ifeq (,$(wildcard $(GITOPS_CONFIG_FILE)))
    $(error gitops config file not found at path: '$(GITOPS_CONFIG_FILE)')
endif
	@oc process -f ./templates/gitops-template.yml --local -p GITOPS_CONFIG='$(shell yq . $(GITOPS_CONFIG_FILE) -r -o=j -I=0)' \
	| oc apply -f - -n $(NAMESPACE)
.PHONY: deploy/gitops

# deploy service via templates to a development Kubernetes/OpenShift cluster
deploy/service: FLEET_MANAGER_IMAGE ?= $(SHORT_IMAGE_REF)
deploy/service: IMAGE_TAG ?= $(image_tag)
deploy/service: FLEET_MANAGER_ENV ?= "development"
deploy/service: REPLICAS ?= "1"
deploy/service: ENABLE_CENTRAL_EXTERNAL_DOMAIN ?= "false"
deploy/service: ENABLE_CENTRAL_LIFE_SPAN ?= "false"
deploy/service: CENTRAL_LIFE_SPAN ?= "48"
deploy/service: OCM_URL ?= "https://api.stage.openshift.com"
deploy/service: SERVICE_PUBLIC_HOST_URL ?= "https://api.openshift.com"
deploy/service: ENABLE_TERMS_ACCEPTANCE ?= "false"
deploy/service: ENABLE_DENY_LIST ?= "false"
deploy/service: ALLOW_EVALUATOR_INSTANCE ?= "true"
deploy/service: QUOTA_TYPE ?= "quota-management-list"
deploy/service: DATAPLANE_CLUSTER_SCALING_TYPE ?= "manual"
deploy/service: CENTRAL_IDP_ISSUER ?= "https://sso.stage.redhat.com/auth/realms/redhat-external"
deploy/service: CENTRAL_IDP_CLIENT_ID ?= "rhacs-ms-dev"
deploy/service: CENTRAL_REQUEST_EXPIRATION_TIMEOUT ?= "1h"
deploy/service: ENABLE_HTTPS ?= "false"
deploy/service: HEALTH_CHECK_SCHEME ?= "HTTP"
deploy/service: CPU_REQUEST ?= "200m"
deploy/service: MEMORY_REQUEST ?= "300Mi"
deploy/service: CPU_LIMIT ?= "200m"
deploy/service: MEMORY_LIMIT ?= "300Mi"
deploy/service: CENTRAL_DOMAIN_NAME ?= "rhacs-dev.com"
deploy/service: KUBERNETES_ISSUER_ENABLED ?= "true"
deploy/service: deploy/envoy deploy/route deploy/gitops
	@time timeout --foreground 3m bash -c "until oc get routes -n $(NAMESPACE) | grep -q fleet-manager; do echo 'waiting for fleet-manager route to be created'; sleep 1; done"
ifeq (,$(wildcard $(PROVIDERS_CONFIG_FILE)))
	$(error providers config file not found at path: '$(PROVIDERS_CONFIG_FILE)')
endif
	@oc process -f ./templates/service-template.yml --local \
		-p ENVIRONMENT="$(FLEET_MANAGER_ENV)" \
		-p CENTRAL_IDP_ISSUER="$(CENTRAL_IDP_ISSUER)" \
		-p CENTRAL_IDP_CLIENT_ID="$(CENTRAL_IDP_CLIENT_ID)" \
		-p REPO_DIGEST="$(FLEET_MANAGER_IMAGE)" \
		-p IMAGE_TAG=$(IMAGE_TAG) \
		-p REPLICAS="${REPLICAS}" \
		-p ENABLE_CENTRAL_EXTERNAL_DOMAIN="${ENABLE_CENTRAL_EXTERNAL_DOMAIN}" \
		-p ENABLE_CENTRAL_LIFE_SPAN="${ENABLE_CENTRAL_LIFE_SPAN}" \
		-p CENTRAL_LIFE_SPAN="${CENTRAL_LIFE_SPAN}" \
		-p ENABLE_OCM_MOCK=$(ENABLE_OCM_MOCK) \
		-p OCM_MOCK_MODE=$(OCM_MOCK_MODE) \
		-p OCM_URL="$(OCM_URL)" \
		-p AMS_URL="${AMS_URL}" \
		-p SERVICE_PUBLIC_HOST_URL="https://$(shell oc get routes/fleet-manager -o jsonpath="{.spec.host}" -n $(NAMESPACE))" \
		-p ENABLE_TERMS_ACCEPTANCE="${ENABLE_TERMS_ACCEPTANCE}" \
		-p ALLOW_EVALUATOR_INSTANCE="${ALLOW_EVALUATOR_INSTANCE}" \
		-p QUOTA_TYPE="${QUOTA_TYPE}" \
		-p DATAPLANE_CLUSTER_SCALING_TYPE="${DATAPLANE_CLUSTER_SCALING_TYPE}" \
		-p CENTRAL_REQUEST_EXPIRATION_TIMEOUT="${CENTRAL_REQUEST_EXPIRATION_TIMEOUT}" \
		-p CLUSTER_LIST='$(shell make -s cluster-list)' \
		-p ENABLE_HTTPS="$(ENABLE_HTTPS)" \
		-p HEALTH_CHECK_SCHEME="$(HEALTH_CHECK_SCHEME)" \
		-p CPU_REQUEST="$(CPU_REQUEST)" \
		-p MEMORY_REQUEST="$(MEMORY_REQUEST)" \
		-p CPU_LIMIT="$(CPU_LIMIT)" \
		-p MEMORY_LIMIT="$(MEMORY_LIMIT)" \
		-p CENTRAL_DOMAIN_NAME="$(CENTRAL_DOMAIN_NAME)" \
		-p SUPPORTED_CLOUD_PROVIDERS='$(shell yq .supported_providers $(PROVIDERS_CONFIG_FILE) -r -o=j -I=0)' \
		-p REGISTERED_USERS_PER_ORGANISATION='$(shell yq .registered_users_per_organisation $(QUOTA_MANAGEMENT_LIST_CONFIG_FILE) -r -o=j -I=0)' \
		-p KUBERNETES_ISSUER_ENABLED="$(KUBERNETES_ISSUER_ENABLED)" \
		| oc apply -f - -n $(NAMESPACE)
.PHONY: deploy/service



# remove service deployments from an OpenShift cluster
undeploy: FLEET_MANAGER_IMAGE ?= $(SHORT_IMAGE_REF)
undeploy:
	@-oc process -f ./templates/db-template.yml --local | oc delete -f - -n $(NAMESPACE)
	@-oc process -f ./templates/secrets-template.yml --local | oc delete -f - -n $(NAMESPACE)
	@-oc process -f ./templates/route-template.yml --local | oc delete -f - -n $(NAMESPACE)
	@-oc delete -f ./templates/envoy-config-configmap.yml -n $(NAMESPACE)
	@-oc process -f ./templates/service-template.yml --local \
		-p REPO_DIGEST="$(FLEET_MANAGER_IMAGE)" \
		| oc delete -f - -n $(NAMESPACE)
.PHONY: undeploy

# Deploys OpenShift ingress router on a k8s cluster
deploy/openshift-router:
	./scripts/openshift-router.sh deploy
.PHONY: deploy/openshift-router

# Un-deploys OpenShift ingress router from a k8s cluster
undeploy/openshift-router:
	./scripts/openshift-router.sh undeploy
.PHONY: undeploy/openshift-router

# Deploys fleet* components with the database on the k8s cluster in use
# Intended for a local / infra cluster deployment and dev testing
deploy/dev:
	./dev/env/scripts/up.sh
.PHONY: deploy/dev

# Un-deploys fleet* components with the database on the k8s cluster in use
undeploy/dev:
	./dev/env/scripts/down.sh
.PHONY: undeploy/dev

# Sets up dev environment by installing the necessary components such as stackrox-operator, openshift-router and other
deploy/bootstrap:
	./dev/env/scripts/bootstrap.sh
.PHONY: deploy/bootstrap

# Deploy local images fast for development
deploy/dev-fast: image/build deploy/dev-fast/fleet-manager deploy/dev-fast/fleetshard-sync

deploy/dev-fast/fleet-manager: image/build
	kubectl -n $(NAMESPACE) set image deploy/fleet-manager service=$(SHORT_IMAGE_REF) migration=$(SHORT_IMAGE_REF)
	kubectl -n $(NAMESPACE) delete pod -l app=fleet-manager

deploy/dev-fast/fleetshard-sync: image/build
	kubectl -n $(NAMESPACE) set image deploy/fleetshard-sync fleetshard-sync=$(SHORT_IMAGE_REF)
	kubectl -n $(NAMESPACE) delete pod -l app=fleetshard-sync

deploy/probe: IMAGE_REGISTRY?="$(external_image_registry)"
deploy/probe: IMAGE_REPOSITORY?="$(probe_image_repository)"
deploy/probe: IMAGE_TAG?="$(image_tag)"
deploy/probe:
	@oc create namespace $(PROBE_NAMESPACE) --dry-run=client -o yaml | oc apply -f -
	@oc create secret generic probe-credentials \
             --save-config \
             --dry-run=client \
             --from-literal=OCM_USERNAME=${USER} \
             --from-literal=OCM_TOKEN='$(shell ocm token --refresh)' \
             -o yaml | oc apply -n $(PROBE_NAMESPACE) -f -
	@oc process -f ./templates/probe-template.yml --local \
		-p IMAGE_REGISTRY=$(IMAGE_REGISTRY) \
		-p IMAGE_REPOSITORY=$(IMAGE_REPOSITORY) \
		-p IMAGE_TAG=$(IMAGE_TAG) \
		| oc apply -f - -n $(PROBE_NAMESPACE)
.PHONY: deploy/probe

undeploy/probe: IMAGE_REGISTRY="$(external_image_registry)"
undeploy/probe: IMAGE_REPOSITORY="$(probe_image_repository)"
undeploy/probe:
	@oc process -f ./templates/probe-template.yml --local \
		-p IMAGE_REGISTRY=$(IMAGE_REGISTRY) \
		-p IMAGE_REPOSITORY=$(IMAGE_REPOSITORY) \
			| oc delete -f - -n $(PROBE_NAMESPACE) --ignore-not-found
	@oc delete secret probe-credentials --ignore-not-found
	@oc delete namespace $(PROBE_NAMESPACE) --ignore-not-found
.PHONY: undeploy/probe

tag:
	@echo "$(image_tag)"
.PHONY: tag

full-image-tag:
	@echo "$(IMAGE_NAME):$(image_tag)"
.PHONY: full-image-tag

image-name/emailsender:
	@echo "$(external_image_registry)/$(emailsender_image_repository)"
.PHONY: image-name/emailsender

image-tag/emailsender:
	@echo "$(external_image_registry)/$(emailsender_image_repository):$(image_tag)"
.PHONY: image-tag/emailsender

clean/go-generated:
	@echo "Cleaning generated .go files..."
	@find . -name '*.go' | xargs grep -l '// Code generated by .*; DO NOT EDIT.$$' | while read -r file; do echo ""$$file""; rm -f "$$file"; done
.PHONY: clean/go-generated

cluster-list:
	@yq '.clusters | .[0].cluster_dns="$(CLUSTER_DNS)"' $(DATAPLANE_CLUSTER_CONFIG_FILE) -r -o=j -I=0
.PHONY: cluster-list

CLUSTER_ID ?= test
run/emailsender:
	@CLUSTER_ID=$(CLUSTER_ID) go run emailsender/cmd/app/main.go
.PHONY: run/emailsender

deploy/emailsender: IMAGE_REPO?="$(external_image_registry)/$(emailsender_image_repository)"
deploy/emailsender: IMAGE_TAG?="$(image_tag)"
deploy/emailsender:
	@kubectl apply -n "$(NAMESPACE)" -f "dev/env/manifests/emailsender-db"
	@helm upgrade --install -n "$(NAMESPACE)" emailsender "deploy/charts/emailsender" \
		--values "dev/env/values/emailsender/values.yaml" \
		--set image.repo="$(IMAGE_REPO)" \
		--set image.tag="$(IMAGE_TAG)"
.PHONY: deploy/emailsender

undeploy/emailsender:
	@helm uninstall -n "$(NAMESPACE)" emailsender --ignore-not-found
	@kubectl delete -n "$(NAMESPACE)" -f "dev/env/manifests/emailsender-db" --ignore-not-found=true
.PHONY: undeploy/emailsender

deploy/fleetshard-sync: FLEET_MANAGER_IMAGE?="$(IMAGE_NAME):$(image_tag)"
deploy/fleetshard-sync: ARGOCD_TENANT_APP_TARGET_REVISION?="HEAD"
deploy/fleetshard-sync: ARGOCD_NAMESPACE?="openshift-gitops"
deploy/fleetshard-sync: MANAGED_DB_ENABLED?="false"
deploy/fleetshard-sync:
	@helm upgrade --install -n "$(NAMESPACE)" fleetshard-sync "deploy/charts/fleetshard-sync" \
		--values "dev/env/values/fleetshard-sync/values.yaml" \
		--set image.ref="$(FLEET_MANAGER_IMAGE)" \
		--set gitops.tenantDefaultAppSourceTargetRevision="$(ARGOCD_TENANT_APP_TARGET_REVISION)" \
		--set argoCdNamespace="$(ARGOCD_NAMESPACE)" \
		--set managedDB.enabled="$(MANAGED_DB_ENABLED)" \
		--set managedDB.subnetGroup="$(MANAGED_DB_SUBNET_GROUP)" \
		--set managedDB.securityGroup="$(MANAGED_DB_SECURITY_GROUP)"
.PHONY: deploy/fleetshard-sync

undeploy/fleetshard-sync:
	@helm uninstall -n "$(NAMESPACE)" fleetshard-sync --ignore-not-found
.PHONY: undeploy/fleetshard-sync
