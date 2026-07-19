ARG WORKSPACE_BASE_IMAGE=ghcr.io/cofy-x/elyro/workspace-base:v0.1.1
FROM ${WORKSPACE_BASE_IMAGE}

ARG DEBIAN_FRONTEND=noninteractive
ARG TARGETARCH
ARG NODE_VERSION=24.18.0
ARG NODE_SHA256_AMD64=55aa7153f9d88f28d765fcdad5ae6945b5c0f98a36881703817e4c450fa76742
ARG NODE_SHA256_ARM64=58c9520501f6ae2b52d5b210444e24b9d0c029a58c5011b797bc1fe7105886f6
ARG ELYRO_PROXY_URL=
ARG ELYRO_PROXY_NO_PROXY=localhost,127.0.0.1,::1,host.docker.internal,.local

USER root
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt/lists,sharing=locked \
    ELYRO_PROXY_URL="${ELYRO_PROXY_URL}" ELYRO_PROXY_NO_PROXY="${ELYRO_PROXY_NO_PROXY}" \
      elyro-proxy-run apt-get update && \
    ELYRO_PROXY_URL="${ELYRO_PROXY_URL}" ELYRO_PROXY_NO_PROXY="${ELYRO_PROXY_NO_PROXY}" \
      elyro-proxy-run apt-get install -y --no-install-recommends g++ make python3

RUN case "${TARGETARCH}" in \
      amd64) NODE_ARCH=x64; NODE_SHA256="${NODE_SHA256_AMD64}" ;; \
      arm64) NODE_ARCH=arm64; NODE_SHA256="${NODE_SHA256_ARM64}" ;; \
      *) echo "unsupported architecture for Node.js: ${TARGETARCH}" >&2; exit 1 ;; \
    esac && \
    NODE_ARCHIVE="node-v${NODE_VERSION}-linux-${NODE_ARCH}.tar.xz" && \
    ELYRO_PROXY_URL="${ELYRO_PROXY_URL}" ELYRO_PROXY_NO_PROXY="${ELYRO_PROXY_NO_PROXY}" \
      elyro-proxy-run curl --retry 3 -fsSLo "/tmp/${NODE_ARCHIVE}" \
        "https://nodejs.org/download/release/v${NODE_VERSION}/${NODE_ARCHIVE}" && \
    echo "${NODE_SHA256}  /tmp/${NODE_ARCHIVE}" | sha256sum --check --strict - && \
    tar -C /usr/local --strip-components=1 -xJf "/tmp/${NODE_ARCHIVE}" && \
    rm -f "/tmp/${NODE_ARCHIVE}" && \
    corepack enable && \
    node --version && npm --version && corepack --version

WORKDIR /home/elyro/workspace
