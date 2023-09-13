
ifeq ($(DEBUG_IMAGE),true)
PROBE_IMAGE_NAME = probe
else
PROBE_IMAGE_NAME = probe-dbg
PROBE_SHORT_IMAGE_REF = "$(PROBE_IMAGE_NAME):$(image_tag)"
endif

probe_image_repository:=$(NAMESPACE)/$(PROBE_IMAGE_NAME)

docker/login/probe:
	@docker logout quay.io
	@DOCKER_CONFIG=${DOCKER_CONFIG} $(DOCKER) login -u "${QUAY_PROBE_USER}" --password-stdin <<< "${QUAY_PROBE_TOKEN}" quay.io
.PHONY: docker/login/probe

image/build/multi-target/probe: GOOS=linux
image/build/multi-target/probe: IMAGE_REF="$(external_image_registry)/$(probe_image_repository):$(image_tag)"
image/build/multi-target/probe:
	DOCKER_CONFIG=${DOCKER_CONFIG} $(DOCKER) build --target $(IMAGE_TARGET) -t $(IMAGE_REF) -f probe/Dockerfile .
	DOCKER_CONFIG=${DOCKER_CONFIG} $(DOCKER) tag $(IMAGE_REF) $(PROBE_SHORT_IMAGE_REF)
.PHONY: image/build/multi-target/probe

image/push/probe: IMAGE_REF="$(external_image_registry)/$(probe_image_repository):$(image_tag)"
image/push/probe: image/build/multi-target/probe
	DOCKER_CONFIG=${DOCKER_CONFIG} $(DOCKER) push $(IMAGE_REF)
	@echo
	@echo "Image was pushed as $(IMAGE_REF)."
.PHONY: image/push/probe

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