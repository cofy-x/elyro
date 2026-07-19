ARG WORKSPACE_BASE_IMAGE=ghcr.io/cofy-x/elyro/workspace-base:v0.1.1
FROM ${WORKSPACE_BASE_IMAGE}

ARG DEBIAN_FRONTEND=noninteractive
ARG TARGETARCH
ARG GO_VERSION=1.25.12
ARG GO_SHA256_AMD64=234828b7a89e0e303d2556310ee549fbcf253d28de937bac3da13d6294262ac1
ARG GO_SHA256_ARM64=8b5884aef89600aef5b0b051fb971f11f49bb996521e911f30f02a66884f7bd2
ARG ELYRO_PROXY_URL=
ARG ELYRO_PROXY_NO_PROXY=localhost,127.0.0.1,::1,host.docker.internal,.local
ARG GOPROXY=https://proxy.golang.org,direct

USER root
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt/lists,sharing=locked \
    ELYRO_PROXY_URL="${ELYRO_PROXY_URL}" ELYRO_PROXY_NO_PROXY="${ELYRO_PROXY_NO_PROXY}" \
      elyro-proxy-run apt-get update && \
    ELYRO_PROXY_URL="${ELYRO_PROXY_URL}" ELYRO_PROXY_NO_PROXY="${ELYRO_PROXY_NO_PROXY}" \
      elyro-proxy-run apt-get install -y --no-install-recommends build-essential

RUN case "${TARGETARCH}" in \
      amd64) GO_ARCH=amd64; GO_SHA256="${GO_SHA256_AMD64}" ;; \
      arm64) GO_ARCH=arm64; GO_SHA256="${GO_SHA256_ARM64}" ;; \
      *) echo "unsupported architecture for Go: ${TARGETARCH}" >&2; exit 1 ;; \
    esac && \
    GO_ARCHIVE="go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" && \
    ELYRO_PROXY_URL="${ELYRO_PROXY_URL}" ELYRO_PROXY_NO_PROXY="${ELYRO_PROXY_NO_PROXY}" \
      elyro-proxy-run curl --retry 3 -fsSLo "/tmp/${GO_ARCHIVE}" "https://go.dev/dl/${GO_ARCHIVE}" && \
    echo "${GO_SHA256}  /tmp/${GO_ARCHIVE}" | sha256sum --check --strict - && \
    rm -rf /usr/local/go && \
    tar -C /usr/local -xzf "/tmp/${GO_ARCHIVE}" && \
    rm -f "/tmp/${GO_ARCHIVE}"

ENV GOPROXY=${GOPROXY}
ENV PATH="/usr/local/go/bin:${PATH}"

WORKDIR /home/elyro/workspace
RUN printf 'PATH="/usr/local/go/bin:/home/elyro/.local/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"\n' >/etc/environment && \
    printf '%s\n' 'export PATH="/usr/local/go/bin:/home/elyro/.local/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"' >/etc/profile.d/elyro-toolchain.sh && \
    go version
