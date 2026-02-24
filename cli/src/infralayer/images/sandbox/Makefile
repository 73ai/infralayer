IMAGE_NAME = ghcr.io/73ai/infralayer-sandbox
LOCAL_IMAGE = infralayer/sandbox:latest

.PHONY: build build-amd64 build-arm64 push push-amd64 push-arm64 all login setup

setup:
	docker buildx create --name multiarch --use --bootstrap 2>/dev/null || docker buildx use multiarch

build:
	docker build -t $(LOCAL_IMAGE) .

build-amd64: setup
	docker buildx build --platform linux/amd64 -t $(LOCAL_IMAGE) --load .

build-arm64: setup
	docker buildx build --platform linux/arm64 -t $(LOCAL_IMAGE) --load .

push: push-amd64 push-arm64
	@echo "Pushed both architectures"

push-amd64: setup
	docker buildx build --platform linux/amd64 \
		-t $(IMAGE_NAME):latest-amd64 \
		--provenance=false \
		--push .

push-arm64: setup
	docker buildx build --platform linux/arm64 \
		-t $(IMAGE_NAME):latest-arm64 \
		--provenance=false \
		--push .

login:
	@echo "Login to GitHub Container Registry"
	@echo "Run: echo \$$GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin"

all: push
