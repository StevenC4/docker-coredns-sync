.PHONY: build build-dev push up down init-env release

PROJECT_NAME := docker-coredns-sync
IMAGE := ghcr.io/$(shell echo $(USER) | tr '[:upper:]' '[:lower:]')/$(PROJECT_NAME)
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo latest)

# Default to prod build
build:
	docker build -t $(IMAGE):$(VERSION) --target prod .

# Dev container build (optional)
build-dev:
	docker build -t $(IMAGE):dev --target dev .

# Push image
push:
	docker push $(IMAGE):$(VERSION)

# Start Docker Compose (prod image)
up:
	docker compose up -d

down:
	docker compose down

# Create empty .env if not present
init-env:
	@if [ ! -f .env ]; then echo "# Created .env file" > .env; fi

# Tag and push for release
release:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is not set"; exit 1; fi
	git tag v$(VERSION)
	git push origin v$(VERSION)
