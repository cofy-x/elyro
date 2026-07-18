FROM ubuntu:24.04@sha256:4fbb8e6a8395de5a7550b33509421a2bafbc0aab6c06ba2cef9ebffbc7092d90

ARG MIRROR_SOURCE=official
ARG ELYRO_PROXY_URL=
ENV DEBIAN_FRONTEND=noninteractive

RUN if [ "${MIRROR_SOURCE}" != "official" ]; then \
      case "${MIRROR_SOURCE}" in \
        aliyun) mirror="mirrors.aliyun.com" ;; \
        ustc) mirror="mirrors.ustc.edu.cn" ;; \
        tuna) mirror="mirrors.tuna.tsinghua.edu.cn" ;; \
        *) printf 'unsupported mirror source: %s\n' "${MIRROR_SOURCE}" >&2; exit 1 ;; \
      esac; \
      case "$(dpkg --print-architecture)" in \
        amd64) ubuntu_path="ubuntu" ;; \
        arm64) ubuntu_path="ubuntu-ports" ;; \
        *) printf 'unsupported architecture: %s\n' "$(dpkg --print-architecture)" >&2; exit 1 ;; \
      esac; \
      if [ -f /etc/apt/sources.list ]; then \
        sed -i -E "s#https?://[^/]+/(ubuntu-ports|ubuntu)#http://${mirror}/${ubuntu_path}#g" /etc/apt/sources.list; \
      fi; \
      if [ -f /etc/apt/sources.list.d/ubuntu.sources ]; then \
        sed -i -E "s#https?://[^/]+/(ubuntu-ports|ubuntu)/?#http://${mirror}/${ubuntu_path}/#g" /etc/apt/sources.list.d/ubuntu.sources; \
      fi; \
      http_proxy="${ELYRO_PROXY_URL}" https_proxy="${ELYRO_PROXY_URL}" \
        apt-get -o APT::Update::Error-Mode=any update; \
      http_proxy="${ELYRO_PROXY_URL}" https_proxy="${ELYRO_PROXY_URL}" \
        apt-get install -y --no-install-recommends ca-certificates; \
    fi \
    && http_proxy="${ELYRO_PROXY_URL}" https_proxy="${ELYRO_PROXY_URL}" \
      apt-get -o APT::Update::Error-Mode=any update \
    && http_proxy="${ELYRO_PROXY_URL}" https_proxy="${ELYRO_PROXY_URL}" \
      apt-get install -y --no-install-recommends \
      bash \
      ca-certificates \
      coreutils \
      curl \
      gawk \
      git \
      openssh-server \
      openssl \
      procps \
      sudo \
    && rm -rf /var/lib/apt/lists/* \
    && groupmod --new-name elyro ubuntu \
    && usermod --login elyro --home /home/elyro --move-home --shell /bin/bash ubuntu \
    && usermod --password "$(openssl passwd -6 "$(openssl rand -hex 32)")" elyro \
    && printf '%s\n' 'elyro ALL=(ALL) NOPASSWD:ALL' >/etc/sudoers.d/91-elyro \
    && chmod 0440 /etc/sudoers.d/91-elyro \
    && install -d -m 0755 /run/sshd /etc/ssh/sshd_config.d \
    && printf '%s\n' \
      'PubkeyAuthentication yes' \
      'PasswordAuthentication no' \
      'KbdInteractiveAuthentication no' \
      'PermitRootLogin no' \
      >/etc/ssh/sshd_config.d/99-elyro-workspace.conf

EXPOSE 22
WORKDIR /home/elyro
USER elyro

CMD ["sudo", "-n", "/bin/bash", "-lc", "ssh-keygen -A && exec /usr/sbin/sshd -D -e"]
