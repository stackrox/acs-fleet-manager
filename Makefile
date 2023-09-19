MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_PATH := $(patsubst %/,%,$(dir $(MKFILE_PATH)))
DOCS_DIR := $(PROJECT_PATH)/docs
TOOLS_DIR := $(PROJECT_PATH)/tools

.DEFAULT_GOAL := help
SHELL = bash

# The details of the application:
binary:=fleet-manager

# The image tag for building and pushing comes from TAG environment variable by default.
# If there is no TAG env than CI_TAG is used instead.
# Otherwise image tag is generated based on git tags.
ifeq ($(TAG),)
ifeq (,$(wildcard CI_TAG))
ifeq ($(IGNORE_REPOSITORY_DIRTINESS),true)
TAG=$(shell git describe --tags --abbrev=10 --long)
else
TAG=$(shell git describe --tags --abbrev=10 --dirty --long)
endif
else
TAG=$(shell cat CI_TAG)
endif
endif
image_tag = $(TAG)

GINKGO_FLAGS ?= -v

# The version needs to be different for each deployment because otherwise the
# cluster will not pull the new image from the internal registry:
version:=$(shell date +%s)

ifeq ($(DEBUG_IMAGE),true)
IMAGE_NAME = fleet-manager-dbg
PROBE_IMAGE_NAME = probe-dbg
IMAGE_TARGET = debug
else
IMAGE_NAME = fleet-manager
PROBE_IMAGE_NAME = probe
IMAGE_TARGET = standard
endif

SHORT_IMAGE_REF = "$(IMAGE_NAME):$(image_tag)"
PROBE_SHORT_IMAGE_REF = "$(PROBE_IMAGE_NAME):$(image_tag)"

IMAGE_REGISTRY ?= quay.io/rhacs-eng
image_repository:=$(IMAGE_NAME)
probe_image_repository:=$(PROBE_IMAGE_NAME)

# In the development environment we are pushing the image directly to the image
# registry inside the development cluster. That registry has a different name
# when it is accessed from outside the cluster and when it is accessed from
# inside the cluster. We need the external name to push the image, and the
# internal name to pull it.
external_image_registry:= $(IMAGE_REGISTRY)
internal_image_registry:=image-registry.openshift-image-registry.svc:5000

# Test image name that will be used for PR checks
test_image:=test/$(IMAGE_NAME)

DOCKER ?= docker
DOCKER_CONFIG ?= "${HOME}/.docker"

# Default Variables
ACSMS_NAMESPACE ?= acsms
ENABLE_OCM_MOCK ?= false
OCM_MOCK_MODE ?= emulate-server
JWKS_URL ?= "https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/certs"
SSO_BASE_URL ?="https://identity.api.stage.openshift.com"
SSO_REALM ?="rhoas" # update your realm here

GO := go
GOFMT := gofmt
# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set).
ifeq (,$(shell $(GO) env GOBIN))
GOBIN=$(shell $(GO) env GOPATH)/bin
else
GOBIN=$(shell $(GO) env GOBIN)
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

GOBINDATA_BIN := $(LOCAL_BIN_PATH)/go-bindata
$(GOBINDATA_BIN): $(TOOLS_DIR)/go.mod $(TOOLS_DIR)/go.sum
	cd $(TOOLS_DIR) && GOBIN=${LOCAL_BIN_PATH} $(GO) install github.com/go-bindata/go-bindata/...

CHAMBER_BIN := $(LOCAL_BIN_PATH)/chamber
$(CHAMBER_BIN): $(TOOLS_DIR)/go.mod $(TOOLS_DIR)/go.sum
	@cd $(TOOLS_DIR) && GOBIN=${LOCAL_BIN_PATH} $(GO) install github.com/segmentio/chamber/v2

GINKGO_BIN := $(LOCAL_BIN_PATH)/ginkgo
$(GINKGO_BIN): $(TOOLS_DIR)/go.mod $(TOOLS_DIR)/go.sum
	@cd $(TOOLS_DIR) && GOBIN=${LOCAL_BIN_PATH} $(GO) install github.com/onsi/ginkgo/v2/ginkgo

TOOLS_VENV_DIR := $(LOCAL_BIN_PATH)/tools_venv
$(TOOLS_VENV_DIR): $(TOOLS_DIR)/requirements.txt
	@set -e; \
	trap "rm -rf $(TOOLS_VENV_DIR)" ERR; \
	python3 -m venv $(TOOLS_VENV_DIR); \
	. $(TOOLS_VENV_DIR)/bin/activate; \
	pip install --upgrade pip==22.3.1; \
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
	@echo "make image/build                 build image (hybrid fast build, respecting IGNORE_REPOSITORY_DIRTINESS)"
	@echo "make image/build/multi-target    build image (containerized, respecting DEBUG_IMAGE and IGNORE_REPOSITORY_DIRTINESS) for local deployment"
	@echo "make image/push                  push image"
	@echo "make setup/git/hooks             setup git hooks"
	@echo "make secrets/touch               touch all required secret files"
	@echo "make centralcert/setup           setup the central TLS certificate used for Managed Central Service"
	@echo "make observatorium/setup         setup observatorium secrets used by CI"
	@echo "make observatorium/token-refresher/setup" setup a local observatorium token refresher
	@echo "make docker/login/internal       login to an openshift cluster image registry"
	@echo "make image/build/push/internal   build and push image to an openshift cluster image registry."
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
		echo "Re-run `make setup/git/hooks` setup step after pre-commit has been installed."; \
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
		./probe/...
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

binary: fleet-manager fleetshard-sync probe acsfleetctl
.PHONY: binary

# Install
install: verify lint
	$(GO) install ./cmd/fleet-manager
.PHONY: install

clean:
	rm -f fleet-manager fleetshard-sync probe/bin/probe
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
		./fleetshard/pkg/cipher/...
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
test/integration/dinosaur: $(GOTESTSUM_BIN)
	$(GOTESTSUM_BIN) --junitfile data/results/fleet-manager-integration-tests.xml --format $(GOTESTSUM_FORMAT) -- -p 1 -ldflags -s -v -timeout $(TEST_TIMEOUT) -count=1 $(TESTFLAGS) \
				./internal/dinosaur/test/integration/...
.PHONY: test/integration/dinosaur

test/integration: test/integration/dinosaur
.PHONY: test/integration

# remove OSD cluster after running tests against real OCM
# requires OCM_OFFLINE_TOKEN env var exported
test/cluster/cleanup:
	./scripts/cleanup_test_cluster.sh
.PHONY: test/cluster/cleanup

test/e2e: $(GINKGO_BIN)
	CLUSTER_ID=1234567890abcdef1234567890abcdef \
	RUN_E2E=true \
	ENABLE_CENTRAL_EXTERNAL_CERTIFICATE=$(ENABLE_CENTRAL_EXTERNAL_CERTIFICATE) \
	$(GINKGO_BIN) -r $(GINKGO_FLAGS) \
		--randomize-suites \
		--fail-on-pending --keep-going \
		--cover --coverprofile=cover.profile \
		--race --trace \
		--json-report=e2e-report.json \
		--timeout=$(TEST_TIMEOUT) \
		--slow-spec-threshold=5m \
		 ./e2e/...
.PHONY: test/e2e

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
.PHONY: openapi/validate

# generate the openapi schema and generated package
openapi/generate: openapi/generate/public openapi/generate/private openapi/generate/admin openapi/generate/rhsso
.PHONY: openapi/generate

openapi/generate/public: $(GOBINDATA_BIN) openapi-generator
	rm -rf internal/dinosaur/pkg/api/public
	$(OPENAPI_GENERATOR) validate -i openapi/fleet-manager.yaml
	$(OPENAPI_GENERATOR) generate -i openapi/fleet-manager.yaml -g go -o internal/dinosaur/pkg/api/public --package-name public -t openapi/templates --ignore-file-override ./.openapi-generator-ignore
	$(GOFMT) -w internal/dinosaur/pkg/api/public

	mkdir -p .generate/openapi
	cp ./openapi/fleet-manager.yaml .generate/openapi
	$(GOBINDATA_BIN) -o ./internal/dinosaur/pkg/generated/bindata.go -pkg generated -mode 420 -modtime 1 -prefix .generate/openapi/ .generate/openapi
	$(GOFMT) -w internal/dinosaur/pkg/generated
	rm -rf .generate/openapi
.PHONY: openapi/generate/public

openapi/generate/private: $(GOBINDATA_BIN) openapi-generator
	rm -rf internal/dinosaur/pkg/api/private
	$(OPENAPI_GENERATOR) validate -i openapi/fleet-manager-private.yaml
	$(OPENAPI_GENERATOR) generate -i openapi/fleet-manager-private.yaml -g go -o internal/dinosaur/pkg/api/private --package-name private -t openapi/templates --ignore-file-override ./.openapi-generator-ignore
	$(GOFMT) -w internal/dinosaur/pkg/api/private
.PHONY: openapi/generate/private

openapi/generate/admin: $(GOBINDATA_BIN) openapi-generator
	rm -rf internal/dinosaur/pkg/api/admin/private
	$(OPENAPI_GENERATOR) validate -i openapi/fleet-manager-private-admin.yaml
	$(OPENAPI_GENERATOR) generate -i openapi/fleet-manager-private-admin.yaml -g go -o internal/dinosaur/pkg/api/admin/private --package-name private -t openapi/templates --ignore-file-override ./.openapi-generator-ignore
	$(GOFMT) -w internal/dinosaur/pkg/api/admin/private
.PHONY: openapi/generate/admin

openapi/generate/rhsso: $(GOBINDATA_BIN) openapi-generator
	rm -rf pkg/client/redhatsso/api
	$(OPENAPI_GENERATOR) validate -i openapi/rh-sso-dynamic-client.yaml
	$(OPENAPI_GENERATOR) generate -i openapi/rh-sso-dynamic-client.yaml -g go -o pkg/client/redhatsso/api --package-name api -t openapi/templates --ignore-file-override ./.openapi-generator-ignore
	$(GOFMT) -w pkg/client/redhatsso/api
.PHONY: openapi/generate/rhsso

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

run: install
	fleet-manager migrate
	fleet-manager serve --public-host-url=${PUBLIC_HOST_URL}
.PHONY: run

# Run Swagger and host the api docs
run/docs:
	$(DOCKER) run -u $(shell id -u) --rm --name swagger_ui_docs -d -p 8082:8080 -e URLS="[ \
		{ url: \"./openapi/fleet-manager.yaml\", name: \"Public API\" },\
		{ url: \"./openapi/fleet-manager-private.yaml\", name: \"Private API\"},\
		{ url: \"./openapi/fleet-manager-private-admin.yaml\", name: \"Private Admin API\"}]"\
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

# Login to docker
docker/login: docker/login/fleet-manager
.PHONY: docker/login

docker/login/fleet-manager:
	@docker logout quay.io
	@DOCKER_CONFIG=${DOCKER_CONFIG} $(DOCKER) login -u "${QUAY_USER}" --password-stdin <<< "${QUAY_TOKEN}" quay.io
.PHONY: docker/login/fleet-manager

docker/login/probe:
	@docker logout quay.io
	@DOCKER_CONFIG=${DOCKER_CONFIG} $(DOCKER) login -u "${QUAY_PROBE_USER}" --password-stdin <<< "${QUAY_PROBE_TOKEN}" quay.io
.PHONY: docker/login/probe

# Login to the OpenShift internal registry
docker/login/internal:
	$(DOCKER) login -u kubeadmin --password-stdin <<< $(shell oc whoami -t) $(shell oc get route default-route -n openshift-image-registry -o jsonpath="{.spec.host}")
.PHONY: docker/login/internal

# Build the image in a hybrid fashion, i.e. building binaries directly on the host leveraging
# Go's cross-compilation capabilities and then copying these binaries into a new Docker image.
image/build: GOOS=linux
image/build: IMAGE_REF ?= "$(external_image_registry)/$(image_repository):$(image_tag)"
image/build: fleet-manager fleetshard-sync
	DOCKER_CONFIG=${DOCKER_CONFIG} $(DOCKER) build -t $(IMAGE_REF) -f Dockerfile.hybrid .
	$(DOCKER) tag $(IMAGE_REF) $(SHORT_IMAGE_REF)
ifeq ("$(CLUSTER_TYPE)","kind")
	kind load docker-image $(SHORT_IMAGE_REF)
endif
.PHONY: image/build

# Build the image using by specifying a specific image target within the Dockerfile.
image/build/multi-target: image/build/multi-target/fleet-manager image/build/multi-target/probe
.PHONY: image/build/multi-target

image/build/multi-target/fleet-manager: GOOS=linux
image/build/multi-target/fleet-manager: IMAGE_REF="$(external_image_registry)/$(image_repository):$(image_tag)"
image/build/multi-target/fleet-manager:
	DOCKER_CONFIG=${DOCKER_CONFIG} $(DOCKER) build --target $(IMAGE_TARGET) -t $(IMAGE_REF) .
	DOCKER_CONFIG=${DOCKER_CONFIG} $(DOCKER) tag $(IMAGE_REF) $(SHORT_IMAGE_REF)
	@echo "New image tag: $(SHORT_IMAGE_REF). You might want to"
	@echo "export FLEET_MANAGER_IMAGE=$(SHORT_IMAGE_REF)"
.PHONY: image/build/multi-target/fleet-manager

image/build/multi-target/probe: GOOS=linux
image/build/multi-target/probe: IMAGE_REF="$(external_image_registry)/$(probe_image_repository):$(image_tag)"
image/build/multi-target/probe:
	DOCKER_CONFIG=${DOCKER_CONFIG} $(DOCKER) build --target $(IMAGE_TARGET) -t $(IMAGE_REF) -f probe/Dockerfile .
	DOCKER_CONFIG=${DOCKER_CONFIG} $(DOCKER) tag $(IMAGE_REF) $(PROBE_SHORT_IMAGE_REF)
.PHONY: image/build/multi-target/probe

image/build/fleet-manager-tools: GOOS=linux
image/build/fleet-manager-tools: IMAGE_REF="$(external_image_registry)/rhacs-eng/fleet-manager-tools:$(image_tag)"
image/build/fleet-manager-tools: fleet-manager fleetshard-sync acsfleetctl
	DOCKER_CONFIG=${DOCKER_CONFIG} $(DOCKER) build -t $(IMAGE_REF) -f Dockerfile.tools .
	DOCKER_CONFIG=${DOCKER_CONFIG} $(DOCKER) tag $(IMAGE_REF) fleet-manager-tools:$(image_tag)
.PHONY: image/build/multi-target/fleet-manager-tools

image/push/fleet-manager-tools: IMAGE_REF="$(external_image_registry)/rhacs-eng/fleet-manager-tools:$(image_tag)"
image/push/fleet-manager-tools: image/build/fleet-manager-tools
	DOCKER_CONFIG=${DOCKER_CONFIG} $(DOCKER) push $(IMAGE_REF)
	@echo
	@echo "Image fleet-manager tools was pushed as $(IMAGE_REF)."
.PHONY: image/push/fleet-manager-tools

# build binary and image and tag image for local deployment
image/build/local: GOOS=linux
image/build/local: IMAGE_REF="$(external_image_registry)/$(image_repository):$(image_tag)"
image/build/local: image/build
	DOCKER_CONFIG=${DOCKER_CONFIG} $(DOCKER) tag $(IMAGE_REF) $(SHORT_IMAGE_REF)
	@echo "New image tag: $(SHORT_IMAGE_REF). You might want to"
	@echo "export FLEET_MANAGER_IMAGE=$(SHORT_IMAGE_REF)"
.PHONY: image/build/local

# Build and push the image
image/push: image/push/fleet-manager image/push/probe
.PHONY: image/push

image/push/fleet-manager: IMAGE_REF="$(external_image_registry)/$(image_repository):$(image_tag)"
image/push/fleet-manager: image/build/multi-target/fleet-manager
	DOCKER_CONFIG=${DOCKER_CONFIG} $(DOCKER) push $(IMAGE_REF)
	@echo
	@echo "Image was pushed as $(IMAGE_REF). You might want to"
	@echo "export FLEET_MANAGER_IMAGE=$(IMAGE_REF)"
.PHONY: image/push/fleet-manager

image/push/probe: IMAGE_REF="$(external_image_registry)/$(probe_image_repository):$(image_tag)"
image/push/probe: image/build/multi-target/probe
	DOCKER_CONFIG=${DOCKER_CONFIG} $(DOCKER) push $(IMAGE_REF)
	@echo
	@echo "Image was pushed as $(IMAGE_REF)."
.PHONY: image/push/probe

# push the image to the OpenShift internal registry
image/push/internal: IMAGE_TAG ?= $(image_tag)
image/push/internal: docker/login/internal
	$(DOCKER) push "$(shell oc get route default-route -n openshift-image-registry -o jsonpath="{.spec.host}")/$(image_repository):$(IMAGE_TAG)"
	$(DOCKER) push "$(shell oc get route default-route -n openshift-image-registry -o jsonpath="{.spec.host}")/$(probe_image_repository):$(IMAGE_TAG)"
.PHONY: image/push/internal

# build and push the image to an OpenShift cluster's internal registry
# namespace used in the image repository must exist on the cluster before running this command. Run `make deploy/project` to create the namespace if not available.
image/build/push/internal: image/push/internal
.PHONY: image/build/push/internal

# Build the binary and test image
image/build/test: binary
	$(DOCKER) build -t "$(test_image)" -f Dockerfile.integration.test .
.PHONY: image/build/test

# Run the test container
test/run: image/build/test
	$(DOCKER) run -u $(shell id -u) --net=host -p 9876:9876 -i "$(test_image)"
.PHONY: test/run

# Run the probe based e2e test in container
test/e2e/probe/run: image/build/multi-target/probe
test/e2e/probe/run: IMAGE_REF="$(external_image_registry)/$(probe_image_repository):$(image_tag)"
test/e2e/probe/run:
	$(DOCKER) run \
	-e QUOTA_TYPE="OCM" \
	-e AUTH_TYPE="OCM" \
	-e PROBE_NAME="e2e-probe-$$$$" \
	-e OCM_USERNAME="${OCM_USERNAME}" \
	-e OCM_TOKEN="${OCM_TOKEN}" \
	-e FLEET_MANAGER_ENDPOINT="${FLEET_MANAGER_ENDPOINT}" \
	--rm $(IMAGE_REF) \
	run
.PHONY: test/e2e/probe/run

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
          secrets/central-tls.crt \
          secrets/central-tls.key \
          secrets/central.idp-client-secret \
          secrets/image-pull.dockerconfigjson \
          secrets/observability-config-access.token \
          secrets/ocm-service.clientId \
          secrets/ocm-service.clientSecret \
          secrets/ocm-service.token \
          secrets/rhsso-logs.clientId \
          secrets/rhsso-logs.clientSecret \
          secrets/rhsso-metrics.clientId \
          secrets/rhsso-metrics.clientSecret \
          secrets/redhatsso-service.clientId \
          secrets/redhatsso-service.clientSecret \
          secrets/sentry.key
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

# Setup for the central broker certificate
centralcert/setup:
	@echo -n "$(CENTRAL_TLS_CERT)" > secrets/central-tls.crt
	@echo -n "$(CENTRAL_TLS_KEY)" > secrets/central-tls.key
.PHONY:centralcert/setup

observatorium/setup:
	@echo -n "$(OBSERVATORIUM_CONFIG_ACCESS_TOKEN)" > secrets/observability-config-access.token;
	@echo -n "$(RHSSO_LOGS_CLIENT_ID)" > secrets/rhsso-logs.clientId;
	@echo -n "$(RHSSO_LOGS_CLIENT_SECRET)" > secrets/rhsso-logs.clientSecret;
	@echo -n "$(RHSSO_METRICS_CLIENT_ID)" > secrets/rhsso-metrics.clientId;
	@echo -n "$(RHSSO_METRICS_CLIENT_SECRET)" > secrets/rhsso-metrics.clientSecret;
.PHONY:observatorium/setup

observatorium/token-refresher/setup: PORT ?= 8085
observatorium/token-refresher/setup: IMAGE_TAG ?= latest
observatorium/token-refresher/setup: ISSUER_URL ?= https://sso.redhat.com/auth/realms/redhat-external
observatorium/token-refresher/setup: OBSERVATORIUM_URL ?= https://observatorium-mst.api.stage.openshift.com/api/metrics/v1/manageddinosaur
observatorium/token-refresher/setup:
	@$(DOCKER) run -d -p ${PORT}:${PORT} \
		--restart always \
		--name observatorium-token-refresher quay.io/rhoas/mk-token-refresher:${IMAGE_TAG} \
		/bin/token-refresher \
		--oidc.issuer-url="${ISSUER_URL}" \
		--url="${OBSERVATORIUM_URL}" \
		--oidc.client-id="${CLIENT_ID}" \
		--oidc.client-secret="${CLIENT_SECRET}" \
		--web.listen=":${PORT}"
	@echo The Observatorium token refresher is now running on 'http://localhost:${PORT}'
.PHONY: observatorium/token-refresher/setup

# OCM login
ocm/login:
	@ocm login --url="$(SERVER_URL)" --token="$(OCM_OFFLINE_TOKEN)"
.PHONY: ocm/login

# Setup OCM_OFFLINE_TOKEN and
# OCM Client ID and Secret should be set only when running inside docker in integration ENV)
ocm/setup: OCM_CLIENT_ID ?= ocm-ams-testing
ocm/setup: OCM_CLIENT_SECRET ?= 8f0c06c5-a558-4a78-a406-02deb1fd3f17
ocm/setup:
	@echo -n "$(OCM_OFFLINE_TOKEN)" > secrets/ocm-service.token
	@echo -n "" > secrets/ocm-service.clientId
	@echo -n "" > secrets/ocm-service.clientSecret
ifeq ($(OCM_ENV), integration)
	@if [[ -n "$(DOCKER_PR_CHECK)" ]]; then echo -n "$(OCM_CLIENT_ID)" > secrets/ocm-service.clientId; echo -n "$(OCM_CLIENT_SECRET)" > secrets/ocm-service.clientSecret; fi;
endif
.PHONY: ocm/setup

# create project where the service will be deployed in an OpenShift cluster
deploy/project:
	@-oc new-project $(NAMESPACE)
.PHONY: deploy/project

# deploy the postgres database required by the service to an OpenShift cluster
deploy/db:
	oc process -f ./templates/db-template.yml | oc apply -f - -n $(NAMESPACE)
	@time timeout --foreground 3m bash -c "until oc get pods -n $(NAMESPACE) | grep fleet-manager-db | grep -v deploy | grep -q Running; do echo 'database is not ready yet'; sleep 10; done"
.PHONY: deploy/db

# deploys the secrets required by the service to an OpenShift cluster
deploy/secrets:
	@oc get service/fleet-manager-db -n $(NAMESPACE) || (echo "Database is not deployed, please run 'make deploy/db'"; exit 1)
	@oc process -f ./templates/secrets-template.yml \
		-p DATABASE_HOST="$(shell oc get service/fleet-manager-db -o jsonpath="{.spec.clusterIP}")" \
		-p OCM_SERVICE_CLIENT_ID="$(shell ([ -s './secrets/ocm-service.clientId' ] && [ -z '${OCM_SERVICE_CLIENT_ID}' ]) && cat ./secrets/ocm-service.clientId || echo '${OCM_SERVICE_CLIENT_ID}')" \
		-p OCM_SERVICE_CLIENT_SECRET="$(shell ([ -s './secrets/ocm-service.clientSecret' ] && [ -z '${OCM_SERVICE_CLIENT_SECRET}' ]) && cat ./secrets/ocm-service.clientSecret || echo '${OCM_SERVICE_CLIENT_SECRET}')" \
		-p OCM_SERVICE_TOKEN="$(shell ([ -s './secrets/ocm-service.token' ] && [ -z '${OCM_SERVICE_TOKEN}' ]) && cat ./secrets/ocm-service.token || echo '${OCM_SERVICE_TOKEN}')" \
		-p SENTRY_KEY="$(shell ([ -s './secrets/sentry.key' ] && [ -z '${SENTRY_KEY}' ]) && cat ./secrets/sentry.key || echo '${SENTRY_KEY}')" \
		-p AWS_ACCESS_KEY="$(shell ([ -s './secrets/aws.accesskey' ] && [ -z '${AWS_ACCESS_KEY}' ]) && cat ./secrets/aws.accesskey || echo '${AWS_ACCESS_KEY}')" \
		-p AWS_ACCOUNT_ID="$(shell ([ -s './secrets/aws.accountid' ] && [ -z '${AWS_ACCOUNT_ID}' ]) && cat ./secrets/aws.accountid || echo '${AWS_ACCOUNT_ID}')" \
		-p AWS_SECRET_ACCESS_KEY="$(shell ([ -s './secrets/aws.secretaccesskey' ] && [ -z '${AWS_SECRET_ACCESS_KEY}' ]) && cat ./secrets/aws.secretaccesskey || echo '${AWS_SECRET_ACCESS_KEY}')" \
		-p ROUTE53_ACCESS_KEY="$(shell ([ -s './secrets/aws.route53accesskey' ] && [ -z '${ROUTE53_ACCESS_KEY}' ]) && cat ./secrets/aws.route53accesskey || echo '${ROUTE53_ACCESS_KEY}')" \
		-p ROUTE53_SECRET_ACCESS_KEY="$(shell ([ -s './secrets/aws.route53secretaccesskey' ] && [ -z '${ROUTE53_SECRET_ACCESS_KEY}' ]) && cat ./secrets/aws.route53secretaccesskey || echo '${ROUTE53_SECRET_ACCESS_KEY}')" \
		-p SSO_CLIENT_ID="$(shell ([ -s './secrets/redhatsso-service.clientId' ] && [ -z '${SSO_CLIENT_ID}' ]) && cat ./secrets/redhatsso-service.clientId || echo '${SSO_CLIENT_ID}')" \
		-p SSO_CLIENT_SECRET="$(shell ([ -s './secrets/redhatsso-service.clientSecret' ] && [ -z '${SSO_CLIENT_SECRET}' ]) && cat ./secrets/redhatsso-service.clientSecret || echo '${SSO_CLIENT_SECRET}')" \
		-p CENTRAL_IDP_CLIENT_SECRET="$(shell ([ -s './secrets/central.idp-client-secret' ] && [ -z '${CENTRAL_IDP_CLIENT_SECRET}' ]) && cat ./secrets/central.idp-client-secret || echo '${CENTRAL_IDP_CLIENT_SECRET}')" \
		-p CENTRAL_TLS_CERT="$(shell ([ -s './secrets/central-tls.crt' ] && [ -z '${CENTRAL_TLS_CERT}' ]) && cat ./secrets/central-tls.crt || echo '${CENTRAL_TLS_CERT}')" \
		-p CENTRAL_TLS_KEY="$(shell ([ -s './secrets/central-tls.key' ] && [ -z '${CENTRAL_TLS_KEY}' ]) && cat ./secrets/central-tls.key || echo '${CENTRAL_TLS_KEY}')" \
		-p OBSERVABILITY_CONFIG_ACCESS_TOKEN="$(shell ([ -s './secrets/observability-config-access.token' ] && [ -z '${OBSERVABILITY_CONFIG_ACCESS_TOKEN}' ]) && cat ./secrets/observability-config-access.token || echo '${OBSERVABILITY_CONFIG_ACCESS_TOKEN}')" \
		-p IMAGE_PULL_DOCKER_CONFIG="$(shell ([ -s './secrets/image-pull.dockerconfigjson' ] && [ -z '${IMAGE_PULL_DOCKER_CONFIG}' ]) && cat ./secrets/image-pull.dockerconfigjson || echo '${IMAGE_PULL_DOCKER_CONFIG}')" \
		-p KUBE_CONFIG="${KUBE_CONFIG}" \
		-p OBSERVABILITY_RHSSO_LOGS_CLIENT_ID="$(shell ([ -s './secrets/rhsso-logs.clientId' ] && [ -z '${OBSERVABILITY_RHSSO_LOGS_CLIENT_ID}' ]) && cat ./secrets/rhsso-logs.clientId || echo '${OBSERVABILITY_RHSSO_LOGS_CLIENT_ID}')" \
		-p OBSERVABILITY_RHSSO_LOGS_SECRET="$(shell ([ -s './secrets/rhsso-logs.clientSecret' ] && [ -z '${OBSERVABILITY_RHSSO_LOGS_SECRET}' ]) && cat ./secrets/rhsso-logs.clientSecret || echo '${OBSERVABILITY_RHSSO_LOGS_SECRET}')" \
		-p OBSERVABILITY_RHSSO_METRICS_CLIENT_ID="$(shell ([ -s './secrets/rhsso-metrics.clientId' ] && [ -z '${OBSERVABILITY_RHSSO_METRICS_CLIENT_ID}' ]) && cat ./secrets/rhsso-metrics.clientId || echo '${OBSERVABILITY_RHSSO_METRICS_CLIENT_ID}')" \
		-p OBSERVABILITY_RHSSO_METRICS_SECRET="$(shell ([ -s './secrets/rhsso-metrics.clientSecret' ] && [ -z '${OBSERVABILITY_RHSSO_METRICS_SECRET}' ]) && cat ./secrets/rhsso-metrics.clientSecret || echo '${OBSERVABILITY_RHSSO_METRICS_SECRET}')" \
		-p OBSERVABILITY_RHSSO_GRAFANA_CLIENT_ID="${OBSERVABILITY_RHSSO_GRAFANA_CLIENT_ID}" \
		-p OBSERVABILITY_RHSSO_GRAFANA_CLIENT_SECRET="${OBSERVABILITY_RHSSO_GRAFANA_CLIENT_SECRET}" \
		| oc apply -f - -n $(NAMESPACE)
.PHONY: deploy/secrets

deploy/envoy:
	@oc apply -f ./templates/envoy-config-configmap.yml -n $(NAMESPACE)
.PHONY: deploy/envoy

deploy/route:
	@oc process -f ./templates/route-template.yml | oc apply -f - -n $(NAMESPACE)
.PHONY: deploy/route

# deploy service via templates to an OpenShift cluster
deploy/service: IMAGE_REGISTRY ?= $(internal_image_registry)
deploy/service: IMAGE_REPOSITORY ?= $(image_repository)
deploy/service: FLEET_MANAGER_ENV ?= "development"
deploy/service: REPLICAS ?= "1"
deploy/service: ENABLE_CENTRAL_EXTERNAL_CERTIFICATE ?= "false"
deploy/service: ENABLE_CENTRAL_LIFE_SPAN ?= "false"
deploy/service: CENTRAL_LIFE_SPAN ?= "48"
deploy/service: OCM_URL ?= "https://api.stage.openshift.com"
deploy/service: SSO_BASE_URL ?= "https://identity.api.stage.openshift.com"
deploy/service: SSO_REALM ?= "rhoas"
deploy/service: MAX_LIMIT_FOR_SSO_GET_CLIENTS ?= "100"
deploy/service: TOKEN_ISSUER_URL ?= "https://sso.redhat.com/auth/realms/redhat-external"
deploy/service: SERVICE_PUBLIC_HOST_URL ?= "https://api.openshift.com"
deploy/service: ENABLE_TERMS_ACCEPTANCE ?= "false"
deploy/service: ENABLE_DENY_LIST ?= "false"
deploy/service: ALLOW_EVALUATOR_INSTANCE ?= "true"
deploy/service: QUOTA_TYPE ?= "quota-management-list"
deploy/service: CENTRAL_OPERATOR_OLM_INDEX_IMAGE ?= "quay.io/osd-addons/managed-central:production-82b42db"
deploy/service: FLEETSHARD_OLM_INDEX_IMAGE ?= "quay.io/osd-addons/fleetshard-operator:production-82b42db"
deploy/service: OBSERVABILITY_CONFIG_REPO ?= "https://api.github.com/repos/bf2fc6cc711aee1a0c2a/observability-resources-mk/contents"
deploy/service: OBSERVABILITY_CONFIG_CHANNEL ?= "resources"
deploy/service: OBSERVABILITY_CONFIG_TAG ?= "main"
deploy/service: DATAPLANE_CLUSTER_SCALING_TYPE ?= "manual"
deploy/service: CENTRAL_OPERATOR_OPERATOR_ADDON_ID ?= "managed-central-qe"
deploy/service: FLEETSHARD_ADDON_ID ?= "fleetshard-operator-qe"
deploy/service: CENTRAL_IDP_ISSUER ?= "https://sso.stage.redhat.com/auth/realms/redhat-external"
deploy/service: CENTRAL_IDP_CLIENT_ID ?= "rhacs-ms-dev"
deploy/service: CENTRAL_REQUEST_EXPIRATION_TIMEOUT ?= "1h"
deploy/service: deploy/envoy deploy/route
	@if test -z "$(IMAGE_TAG)"; then echo "IMAGE_TAG was not specified"; exit 1; fi
	@time timeout --foreground 3m bash -c "until oc get routes -n $(NAMESPACE) | grep -q fleet-manager; do echo 'waiting for fleet-manager route to be created'; sleep 1; done"
	@oc process -f ./templates/service-template.yml \
		-p ENVIRONMENT="$(FLEET_MANAGER_ENV)" \
		-p CENTRAL_IDP_ISSUER="$(CENTRAL_IDP_ISSUER)" \
		-p CENTRAL_IDP_CLIENT_ID="$(CENTRAL_IDP_CLIENT_ID)" \
		-p IMAGE_REGISTRY=$(IMAGE_REGISTRY) \
		-p IMAGE_REPOSITORY=$(IMAGE_REPOSITORY) \
		-p IMAGE_TAG=$(IMAGE_TAG) \
		-p REPLICAS="${REPLICAS}" \
		-p ENABLE_CENTRAL_EXTERNAL_CERTIFICATE="${ENABLE_CENTRAL_EXTERNAL_CERTIFICATE}" \
		-p ENABLE_CENTRAL_LIFE_SPAN="${ENABLE_CENTRAL_LIFE_SPAN}" \
		-p CENTRAL_LIFE_SPAN="${CENTRAL_LIFE_SPAN}" \
		-p ENABLE_OCM_MOCK=$(ENABLE_OCM_MOCK) \
		-p OCM_MOCK_MODE=$(OCM_MOCK_MODE) \
		-p OCM_URL="$(OCM_URL)" \
		-p AMS_URL="${AMS_URL}" \
		-p JWKS_URL="$(JWKS_URL)" \
		-p SSO_BASE_URL="$(SSO_BASE_URL)" \
		-p SSO_REALM="$(SSO_REALM)" \
		-p MAX_LIMIT_FOR_SSO_GET_CLIENTS="${MAX_LIMIT_FOR_SSO_GET_CLIENTS}" \
		-p TOKEN_ISSUER_URL="${TOKEN_ISSUER_URL}" \
		-p SERVICE_PUBLIC_HOST_URL="https://$(shell oc get routes/fleet-manager -o jsonpath="{.spec.host}" -n $(NAMESPACE))" \
		-p OBSERVATORIUM_RHSSO_GATEWAY="${OBSERVATORIUM_RHSSO_GATEWAY}" \
		-p OBSERVATORIUM_RHSSO_REALM="${OBSERVATORIUM_RHSSO_REALM}" \
		-p OBSERVATORIUM_RHSSO_TENANT="${OBSERVATORIUM_RHSSO_TENANT}" \
		-p OBSERVATORIUM_RHSSO_AUTH_SERVER_URL="${OBSERVATORIUM_RHSSO_AUTH_SERVER_URL}" \
		-p OBSERVATORIUM_TOKEN_REFRESHER_URL="http://token-refresher.$(NAMESPACE).svc.cluster.local" \
		-p OBSERVABILITY_CONFIG_REPO="${OBSERVABILITY_CONFIG_REPO}" \
		-p OBSERVABILITY_CONFIG_TAG="${OBSERVABILITY_CONFIG_TAG}" \
		-p ENABLE_TERMS_ACCEPTANCE="${ENABLE_TERMS_ACCEPTANCE}" \
		-p ALLOW_EVALUATOR_INSTANCE="${ALLOW_EVALUATOR_INSTANCE}" \
		-p QUOTA_TYPE="${QUOTA_TYPE}" \
		-p FLEETSHARD_OLM_INDEX_IMAGE="${FLEETSHARD_OLM_INDEX_IMAGE}" \
		-p CENTRAL_OPERATOR_OLM_INDEX_IMAGE="${CENTRAL_OPERATOR_OLM_INDEX_IMAGE}" \
		-p CENTRAL_OPERATOR_OPERATOR_ADDON_ID="${CENTRAL_OPERATOR_OPERATOR_ADDON_ID}" \
		-p FLEETSHARD_ADDON_ID="${FLEETSHARD_ADDON_ID}" \
		-p DATAPLANE_CLUSTER_SCALING_TYPE="${DATAPLANE_CLUSTER_SCALING_TYPE}" \
		-p CENTRAL_REQUEST_EXPIRATION_TIMEOUT="${CENTRAL_REQUEST_EXPIRATION_TIMEOUT}" \
		| oc apply -f - -n $(NAMESPACE)
.PHONY: deploy/service



# remove service deployments from an OpenShift cluster
undeploy: IMAGE_REGISTRY ?= $(internal_image_registry)
undeploy: IMAGE_REPOSITORY ?= $(image_repository)
undeploy:
	@-oc process -f ./templates/observatorium-token-refresher.yml | oc delete -f - -n $(NAMESPACE)
	@-oc process -f ./templates/db-template.yml | oc delete -f - -n $(NAMESPACE)
	@-oc process -f ./templates/secrets-template.yml | oc delete -f - -n $(NAMESPACE)
	@-oc process -f ./templates/route-template.yml | oc delete -f - -n $(NAMESPACE)
	@-oc delete -f ./templates/envoy-config-configmap.yml -n $(NAMESPACE)
	@-oc process -f ./templates/service-template.yml \
		-p IMAGE_REGISTRY=$(IMAGE_REGISTRY) \
		-p IMAGE_REPOSITORY=$(IMAGE_REPOSITORY) \
		| oc delete -f - -n $(NAMESPACE)
.PHONY: undeploy

# Deploys an Observatorium token refresher on an OpenShift cluster
deploy/token-refresher: ISSUER_URL ?= "https://sso.redhat.com/auth/realms/redhat-external"
deploy/token-refresher: OBSERVATORIUM_TOKEN_REFRESHER_IMAGE ?= "quay.io/rhoas/mk-token-refresher"
deploy/token-refresher: OBSERVATORIUM_TOKEN_REFRESHER_IMAGE_TAG ?= "latest"
deploy/token-refresher: OBSERVATORIUM_URL ?= "https://observatorium-mst.api.stage.openshift.com/api/metrics/v1/manageddinosaur"
deploy/token-refresher:
	@-oc process -f ./templates/observatorium-token-refresher.yml \
		-p ISSUER_URL=${ISSUER_URL} \
		-p OBSERVATORIUM_URL=${OBSERVATORIUM_URL} \
		-p OBSERVATORIUM_TOKEN_REFRESHER_IMAGE=${OBSERVATORIUM_TOKEN_REFRESHER_IMAGE} \
		-p OBSERVATORIUM_TOKEN_REFRESHER_IMAGE_TAG=${OBSERVATORIUM_TOKEN_REFRESHER_IMAGE_TAG} \
		 | oc apply -f - -n $(NAMESPACE)
.PHONY: deploy/token-refresher

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
	kubectl -n $(ACSMS_NAMESPACE) set image deploy/fleet-manager fleet-manager=$(SHORT_IMAGE_REF) db-migrate=$(SHORT_IMAGE_REF)
	kubectl -n $(ACSMS_NAMESPACE) delete pod -l application=fleet-manager

deploy/dev-fast/fleetshard-sync: image/build
	kubectl -n $(ACSMS_NAMESPACE) set image deploy/fleetshard-sync fleetshard-sync=$(SHORT_IMAGE_REF)
	kubectl -n $(ACSMS_NAMESPACE) delete pod -l application=fleetshard-sync

# When making changes to the gitops configuration for development purposes
# situated here dev/env/manifests/fleet-manager/04-gitops-config.yaml, this
# target will update the gitops configmap on the dev cluster.
# It might take a few seconds/minutes for fleet-manager to observe the changes.
# Changes to the configmap are hot-reloaded
# See https://kubernetes.io/docs/concepts/configuration/configmap/#mounted-configmaps-are-updated-automatically
deploy/dev/update-gitops-config:
	./dev/env/scripts/update_gitops_config.sh
.PHONY: deploy/dev/update-gitops-config

tag:
	@echo "$(image_tag)"
.PHONY: tag

full-image-tag:
	@echo "$(IMAGE_NAME):$(image_tag)"
.PHONY: full-image-tag

release_date="$(shell date '+%Y-%m-%d')"
release_commit="$(shell git rev-parse --short=7 HEAD)"
tag_count="$(shell git tag -l $(release_date)* | wc -l | xargs)" # use xargs to remove unnecessary whitespace
start_rev=1
rev="$(shell expr $(tag_count) + $(start_rev))"
release-version:
	@echo "$(release_date).$(rev).$(release_commit)"
.PHONY: release-version
