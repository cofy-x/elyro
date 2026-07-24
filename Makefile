.PHONY: image-report image-report-test elyro-build elyro-install elyro-smoke \
	release-config-check release-install-smoke core-images-smoke \
	workspace-base-image-build workspace-python-image-build workspace-go-image-build workspace-node-image-build \
	workspace-base-image-build-amd64 workspace-base-image-build-arm64 \
	workspace-python-image-build-amd64 workspace-python-image-build-arm64 \
	workspace-go-image-build-amd64 workspace-go-image-build-arm64 \
	workspace-node-image-build-amd64 workspace-node-image-build-arm64 \
	workspace-build workspace-install workspace-smoke workspace-e2e demo-record help list-targets

VERSION ?= dev

core-images-smoke:
	@scripts/ci/build-core-images.sh
	@scripts/ci/smoke-core-images.sh

include mk/common.mk
include mk/help.mk
include mk/apps.mk
include mk/docker-base-images.mk

demo-record: elyro-build
	@command -v vhs >/dev/null 2>&1 || { echo "Missing vhs. Install it from https://github.com/charmbracelet/vhs"; exit 1; }
	@HOME=/tmp/elyro-terminal-demo-home XDG_STATE_HOME=/tmp/elyro-terminal-demo-home/state \
		$(ELYRO_BIN) --project-dir /tmp/elyro-terminal-demo down >/dev/null 2>&1 || true
	@rm -rf /tmp/elyro-terminal-demo /tmp/elyro-terminal-demo-home && mkdir -p /tmp/elyro-terminal-demo /tmp/elyro-terminal-demo-home
	@HOME=/tmp/elyro-terminal-demo-home XDG_STATE_HOME=/tmp/elyro-terminal-demo-home/state \
		NO_COLOR= TERM=xterm-256color CI= PATH="$(CURDIR)/$(ELYRO_BIN_DIR):$$PATH" vhs scripts/demo/elyro.tape
