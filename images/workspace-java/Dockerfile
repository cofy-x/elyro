ARG WORKSPACE_BASE_IMAGE=ghcr.io/cofy-x/elyro/workspace-base:v0.1.1
FROM ${WORKSPACE_BASE_IMAGE}

ARG ELYRO_PROXY_URL=
ARG ELYRO_PROXY_NO_PROXY=localhost,127.0.0.1,::1,host.docker.internal,.local
ARG GRADLE_VERSION=9.6.1
ARG GRADLE_SHA256=9c0f7faeeb306cb14e4279a3e084ca6b596894089a0638e68a07c945a32c9e14

RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt/lists,sharing=locked \
    ELYRO_PROXY_URL="${ELYRO_PROXY_URL}" ELYRO_PROXY_NO_PROXY="${ELYRO_PROXY_NO_PROXY}" \
    elyro-proxy-run apt-get update && \
    ELYRO_PROXY_URL="${ELYRO_PROXY_URL}" ELYRO_PROXY_NO_PROXY="${ELYRO_PROXY_NO_PROXY}" \
    elyro-proxy-run apt-get install -y --no-install-recommends \
      openjdk-21-jdk-headless \
      maven \
      unzip && \
    rm -rf /tmp/* /var/tmp/*

RUN ELYRO_PROXY_URL="${ELYRO_PROXY_URL}" ELYRO_PROXY_NO_PROXY="${ELYRO_PROXY_NO_PROXY}" \
      elyro-proxy-run curl --retry 3 -fsSLo /tmp/gradle.zip \
        "https://services.gradle.org/distributions/gradle-${GRADLE_VERSION}-bin.zip" && \
    echo "${GRADLE_SHA256}  /tmp/gradle.zip" | sha256sum --check --strict - && \
    unzip -q /tmp/gradle.zip -d /opt && \
    ln -s "/opt/gradle-${GRADLE_VERSION}/bin/gradle" /usr/local/bin/gradle && \
    rm -f /tmp/gradle.zip

WORKDIR /home/elyro/workspace

RUN java -version && mvn --version >/dev/null && gradle --version >/dev/null
