.PHONY: image-report image-report-test \
	workspace-base-image-build workspace-python-image-build workspace-go-image-build workspace-node-image-build \
	workspace-base-image-build-amd64 workspace-base-image-build-arm64 \
	workspace-python-image-build-amd64 workspace-python-image-build-arm64 \
	workspace-go-image-build-amd64 workspace-go-image-build-arm64 \
	workspace-node-image-build-amd64 workspace-node-image-build-arm64

ELYRO_PROXY_URL ?=
ELYRO_PROXY_NO_PROXY ?= localhost,127.0.0.1,::1,host.docker.internal,.local
ELYRO_PROXY_RESOLVER ?= ./scripts/lib/resolve-elyro-proxy.sh
ELYRO_MIRROR_SOURCE ?= official
ELYRO_GOPROXY ?= https://proxy.golang.org,direct
export ELYRO_PROXY_URL ELYRO_PROXY_NO_PROXY ELYRO_MIRROR_SOURCE ELYRO_GOPROXY

ELYRO_IMAGE_PREFIX ?= ghcr.io/cofy-x/elyro
FORMAT ?= table
TOP_LAYERS ?= 0

image-report:
	@set -- --version "$(VERSION)" --format "$(FORMAT)" --image-prefix "$(ELYRO_IMAGE_PREFIX)" --top-layers "$(TOP_LAYERS)"; \
	if [ -n "$(COMPARE_VERSION)" ]; then set -- "$$@" --compare-version "$(COMPARE_VERSION)"; fi; \
	go run ./scripts/images/report "$$@"

image-report-test:
	@go test ./scripts/images/report

WORKSPACE_BASE_IMAGE_REPO ?= elyro/workspace-base
WORKSPACE_BASE_PLATFORM ?= linux/amd64
WORKSPACE_PYTHON_IMAGE_REPO ?= elyro/workspace-python
WORKSPACE_PYTHON_PLATFORM ?= linux/amd64
WORKSPACE_GO_IMAGE_REPO ?= elyro/workspace-go
WORKSPACE_GO_PLATFORM ?= linux/amd64
WORKSPACE_NODE_IMAGE_REPO ?= elyro/workspace-node
WORKSPACE_NODE_PLATFORM ?= linux/amd64

define build_workspace_image
	@VERSION_INPUT="$(VERSION)"; [ -n "$$VERSION_INPUT" ] || VERSION_INPUT=dev; \
	PLATFORM_INPUT="$($(1)_PLATFORM)"; ARCH_SUFFIX="$${PLATFORM_INPUT##*/}"; \
	PROXY_URL="$$(bash "$(ELYRO_PROXY_RESOLVER)" "$(ELYRO_PROXY_URL)")"; \
	echo "Building $($(1)_IMAGE_REPO):$$VERSION_INPUT for $$PLATFORM_INPUT"; \
	DOCKER_BUILDKIT=1 docker build --platform "$$PLATFORM_INPUT" \
		--build-arg "MIRROR_SOURCE=$(ELYRO_MIRROR_SOURCE)" \
		--build-arg "ELYRO_PROXY_URL=$$PROXY_URL" \
		--build-arg "ELYRO_PROXY_NO_PROXY=$(ELYRO_PROXY_NO_PROXY)" $(2) \
		-f "images/$(3)/Dockerfile" -t "$($(1)_IMAGE_REPO):$$VERSION_INPUT-$$ARCH_SUFFIX" "images/$(3)"
endef

workspace-base-image-build:
	$(call build_workspace_image,WORKSPACE_BASE,,workspace-base)

workspace-base-image-build-amd64:
	@$(MAKE) workspace-base-image-build WORKSPACE_BASE_PLATFORM=linux/amd64

workspace-base-image-build-arm64:
	@$(MAKE) workspace-base-image-build WORKSPACE_BASE_PLATFORM=linux/arm64

workspace-python-image-build: workspace-base-image-build
	$(call build_workspace_image,WORKSPACE_PYTHON,--build-arg "WORKSPACE_BASE_IMAGE=$(WORKSPACE_BASE_IMAGE_REPO):$${VERSION_INPUT}-$${ARCH_SUFFIX}",workspace-python)

workspace-python-image-build-amd64:
	@$(MAKE) workspace-python-image-build WORKSPACE_PYTHON_PLATFORM=linux/amd64 WORKSPACE_BASE_PLATFORM=linux/amd64

workspace-python-image-build-arm64:
	@$(MAKE) workspace-python-image-build WORKSPACE_PYTHON_PLATFORM=linux/arm64 WORKSPACE_BASE_PLATFORM=linux/arm64

workspace-go-image-build: workspace-base-image-build
	$(call build_workspace_image,WORKSPACE_GO,--build-arg "WORKSPACE_BASE_IMAGE=$(WORKSPACE_BASE_IMAGE_REPO):$${VERSION_INPUT}-$${ARCH_SUFFIX}" --build-arg "GOPROXY=$(ELYRO_GOPROXY)",workspace-go)

workspace-go-image-build-amd64:
	@$(MAKE) workspace-go-image-build WORKSPACE_GO_PLATFORM=linux/amd64 WORKSPACE_BASE_PLATFORM=linux/amd64

workspace-go-image-build-arm64:
	@$(MAKE) workspace-go-image-build WORKSPACE_GO_PLATFORM=linux/arm64 WORKSPACE_BASE_PLATFORM=linux/arm64

workspace-node-image-build: workspace-base-image-build
	$(call build_workspace_image,WORKSPACE_NODE,--build-arg "WORKSPACE_BASE_IMAGE=$(WORKSPACE_BASE_IMAGE_REPO):$${VERSION_INPUT}-$${ARCH_SUFFIX}",workspace-node)

workspace-node-image-build-amd64:
	@$(MAKE) workspace-node-image-build WORKSPACE_NODE_PLATFORM=linux/amd64 WORKSPACE_BASE_PLATFORM=linux/amd64

workspace-node-image-build-arm64:
	@$(MAKE) workspace-node-image-build WORKSPACE_NODE_PLATFORM=linux/arm64 WORKSPACE_BASE_PLATFORM=linux/arm64
