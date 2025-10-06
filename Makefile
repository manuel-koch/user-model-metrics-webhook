THIS_DIR             := $(realpath $(dir $(abspath $(firstword $(MAKEFILE_LIST)))))
IMAGE_GOLANG_VERSION := $(shell grep -E "go\s+\d+\..+" go.mod | head -n1 | cut -d" " -f2)
IMAGE_TAGGED_LATEST  := user-model-metrics-webhook:latest
LOCAL_PORT           := 18500

user-model-metrics-webhook: *.go go.mod
	go build -gcflags="-N -l" .

debug-user-model-metrics-webhook:: user-model-metrics-webhook
	dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec user-model-metrics-webhook

build-image::
	docker build \
		--build-arg GOLANG_VERSION=$(IMAGE_GOLANG_VERSION) \
		-t $(IMAGE_TAGGED_LATEST) \
		-f $(THIS_DIR)/Dockerfile \
		.

run-image::
	docker run --rm \
	-v $(THIS_DIR)/data:/app/data \
	-p $(LOCAL_PORT):80 \
	$(IMAGE_TAGGED_LATEST)
