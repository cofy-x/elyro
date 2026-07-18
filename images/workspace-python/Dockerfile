ARG WORKSPACE_BASE_IMAGE=ghcr.io/cofy-x/elyro/workspace-base:v0.9.0
FROM ${WORKSPACE_BASE_IMAGE}

ARG TARGETARCH
ARG ELYRO_PROXY_URL=
ARG ELYRO_PROXY_NO_PROXY=localhost,127.0.0.1,::1,host.docker.internal,.local
ARG UV_VERSION=0.11.28
ARG UV_SHA256_AMD64=e490a6464492183c5d4534a5527fb4440f7f2bb2f228162ad7e4afe076dc0224
ARG UV_SHA256_ARM64=03e9fe0a81b0718d0bc84625de3885df6cc3f89a8b6af6121d6b9f6113fb6533

USER root
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt/lists,sharing=locked \
    ELYRO_PROXY_URL="${ELYRO_PROXY_URL}" ELYRO_PROXY_NO_PROXY="${ELYRO_PROXY_NO_PROXY}" \
      elyro-proxy-run apt-get update && \
    ELYRO_PROXY_URL="${ELYRO_PROXY_URL}" ELYRO_PROXY_NO_PROXY="${ELYRO_PROXY_NO_PROXY}" \
      elyro-proxy-run apt-get install -y --no-install-recommends \
        build-essential \
        python3 \
        python3-dev \
        python3-venv

RUN case "${TARGETARCH}" in \
      amd64) UV_ARCH=x86_64; UV_SHA256="${UV_SHA256_AMD64}" ;; \
      arm64) UV_ARCH=aarch64; UV_SHA256="${UV_SHA256_ARM64}" ;; \
      *) echo "unsupported architecture for uv: ${TARGETARCH}" >&2; exit 1 ;; \
    esac && \
    UV_ARCHIVE="uv-${UV_ARCH}-unknown-linux-gnu.tar.gz" && \
    ELYRO_PROXY_URL="${ELYRO_PROXY_URL}" ELYRO_PROXY_NO_PROXY="${ELYRO_PROXY_NO_PROXY}" \
      elyro-proxy-run curl --retry 3 -fsSLo "/tmp/${UV_ARCHIVE}" \
        "https://github.com/astral-sh/uv/releases/download/${UV_VERSION}/${UV_ARCHIVE}" && \
    echo "${UV_SHA256}  /tmp/${UV_ARCHIVE}" | sha256sum --check --strict - && \
    tar -xzf "/tmp/${UV_ARCHIVE}" --strip-components=1 -C /usr/local/bin && \
    rm -f "/tmp/${UV_ARCHIVE}" && \
    ln -sfn /usr/bin/python3 /usr/local/bin/python

USER elyro
WORKDIR /home/elyro/workspace
RUN uv --version && \
    uv venv /tmp/elyro-uv-venv && \
    python -m venv /tmp/elyro-python-venv && \
    rm -rf /tmp/elyro-uv-venv /tmp/elyro-python-venv

USER root
