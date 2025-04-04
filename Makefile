.PHONY: build build-dev push up down init-env release unrelease dev-init

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

unrelease:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is not set"; exit 1; fi
	git tag -d v$(VERSION)
	git push --delete origin v$(VERSION)

dev-init:
	@mkdir -p .devcontainer
	@if [ ! -f .devcontainer/.env ]; then \
		UNAME_S=$$(uname -s); \
		if [ "$$UNAME_S" = "Darwin" ]; then \
			HOST_IP=$$(ipconfig getifaddr en0); \
		else \
			HOST_IP=$$(ip route get 1 | awk '{print $$NF; exit}'); \
		fi; \
		echo "ETCD_HOST=http://etcd" >> .devcontainer/.env; \
		echo "HOST_IP=$$HOST_IP" > .devcontainer/.env; \
		echo "HOSTNAME=$$(hostname)" >> .devcontainer/.env; \
		echo "LOG_LEVEL=DEBUG" >> .devcontainer/.env; \
		echo "Created .devcontainer/.env with HOST_IP=$$HOST_IP"; \
	else \
		echo ".devcontainer/.env already exists. Skipping."; \
	fi