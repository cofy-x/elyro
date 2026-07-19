FROM ubuntu:24.04@sha256:4fbb8e6a8395de5a7550b33509421a2bafbc0aab6c06ba2cef9ebffbc7092d90

ARG MIRROR_SOURCE=official
ARG APT_RETRIES=3
ARG APT_TIMEOUT=30
ARG ELYRO_PROXY_URL=
ARG ELYRO_PROXY_NO_PROXY=localhost,127.0.0.1,::1,host.docker.internal,.local
ARG DEBIAN_FRONTEND=noninteractive

RUN ARCH="$(dpkg --print-architecture)" && \
    if [ "$ARCH" = "amd64" ]; then UBUNTU_PATH="ubuntu"; DEFAULT_MIRROR="archive.ubuntu.com"; \
    elif [ "$ARCH" = "arm64" ]; then UBUNTU_PATH="ubuntu-ports"; DEFAULT_MIRROR="ports.ubuntu.com"; \
    else echo "unsupported architecture: $ARCH" >&2; exit 1; fi && \
    if [ "$MIRROR_SOURCE" = "official" ]; then MIRROR="$DEFAULT_MIRROR"; \
    elif [ "$MIRROR_SOURCE" = "aliyun" ]; then MIRROR="mirrors.aliyun.com"; \
    elif [ "$MIRROR_SOURCE" = "ustc" ]; then MIRROR="mirrors.ustc.edu.cn"; \
    elif [ "$MIRROR_SOURCE" = "tuna" ]; then MIRROR="mirrors.tuna.tsinghua.edu.cn"; \
    else echo "unsupported mirror source: $MIRROR_SOURCE" >&2; exit 1; fi && \
    MIRROR_SCHEME="http" && \
    if [ -f /etc/apt/sources.list ]; then \
      sed -i "s|archive.ubuntu.com|$MIRROR|g; s|security.ubuntu.com|$MIRROR|g; s|ports.ubuntu.com|$MIRROR|g; s|http://$MIRROR|$MIRROR_SCHEME://$MIRROR|g" /etc/apt/sources.list; \
    fi && \
    if [ -f /etc/apt/sources.list.d/ubuntu.sources ]; then \
      BASE_URI="$MIRROR_SCHEME://$MIRROR/$UBUNTU_PATH/"; \
      sed -i "s|http://archive.ubuntu.com/ubuntu/|$BASE_URI|g; s|http://security.ubuntu.com/ubuntu/|$BASE_URI|g; s|http://ports.ubuntu.com/ubuntu-ports/|$BASE_URI|g" /etc/apt/sources.list.d/ubuntu.sources; \
    fi

RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt/lists,sharing=locked \
    if [ "$MIRROR_SOURCE" != "official" ]; then \
      http_proxy="${ELYRO_PROXY_URL}" https_proxy="${ELYRO_PROXY_URL}" \
        apt-get -o "Acquire::Retries=${APT_RETRIES}" -o "Acquire::http::Timeout=${APT_TIMEOUT}" -o APT::Update::Error-Mode=any update && \
      http_proxy="${ELYRO_PROXY_URL}" https_proxy="${ELYRO_PROXY_URL}" \
        apt-get install -y --no-install-recommends ca-certificates; \
    fi

COPY --chmod=0755 elyro-apt-update.sh /usr/local/sbin/elyro-apt-update

RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt/lists,sharing=locked \
    ELYRO_APT_RETRIES="${APT_RETRIES}" ELYRO_APT_TIMEOUT="${APT_TIMEOUT}" elyro-apt-update && \
    http_proxy="${ELYRO_PROXY_URL}" https_proxy="${ELYRO_PROXY_URL}" \
      apt-get install -y --no-install-recommends \
        ca-certificates curl git jq locales openssh-server procps sudo tini xz-utils zsh zsh-autosuggestions zsh-syntax-highlighting && \
    rm -rf /tmp/* /var/tmp/*

ENV APP_USER=elyro \
    LANG=en_US.UTF-8 \
    LC_ALL=en_US.UTF-8 \
    PATH=/home/elyro/.local/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

RUN locale-gen en_US.UTF-8 && \
    if getent passwd 1000 >/dev/null; then userdel --remove "$(getent passwd 1000 | cut -d: -f1)"; fi && \
    if getent group 1000 >/dev/null; then groupdel "$(getent group 1000 | cut -d: -f1)"; fi && \
    groupadd --gid 1000 elyro && \
    useradd --uid 1000 --gid 1000 -m -s /usr/bin/zsh elyro && \
    usermod -aG sudo elyro && \
    printf 'elyro ALL=(ALL) NOPASSWD:ALL\n' >/etc/sudoers.d/91-elyro && \
    chmod 0440 /etc/sudoers.d/91-elyro && \
    install -d -m 0755 /run/sshd /home/elyro/workspace && \
    chown -R elyro:elyro /home/elyro && \
    sed -i 's/^AcceptEnv/#AcceptEnv/g' /etc/ssh/sshd_config && \
    printf 'PasswordAuthentication no\nPermitRootLogin no\n' >/etc/ssh/sshd_config.d/00-elyro-security.conf && \
    sed 's@session\s*required\s*pam_loginuid.so@session optional pam_loginuid.so@g' -i /etc/pam.d/sshd && \
    printf 'PATH="%s"\n' "$PATH" >/etc/environment && \
    printf '%s\n' 'export PATH="/home/elyro/.local/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"' >/etc/profile.d/elyro-path.sh && \
    chmod 0644 /etc/profile.d/elyro-path.sh

RUN cat >/usr/local/bin/elyro-proxy-run <<'EOF_PROXY_RUN' && chmod 0755 /usr/local/bin/elyro-proxy-run
#!/usr/bin/env bash
set -euo pipefail
proxy_url="${ELYRO_PROXY_URL:-}"
no_proxy_value="${ELYRO_PROXY_NO_PROXY:-localhost,127.0.0.1,::1,host.docker.internal,.local}"
if [[ -n "${proxy_url}" ]]; then
  export http_proxy="${proxy_url}" https_proxy="${proxy_url}" all_proxy="${proxy_url}"
  export HTTP_PROXY="${proxy_url}" HTTPS_PROXY="${proxy_url}" ALL_PROXY="${proxy_url}"
  export no_proxy="${no_proxy_value}" NO_PROXY="${no_proxy_value}"
fi
exec "$@"
EOF_PROXY_RUN

COPY --chown=elyro:elyro --chmod=0644 elyro.zshrc /home/elyro/.zshrc

USER root
WORKDIR /home/elyro/workspace
EXPOSE 22
ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["/usr/sbin/sshd", "-D", "-e"]
