ARG WORKSPACE_GO_IMAGE=elyro/workspace-go:dev-amd64
FROM ${WORKSPACE_GO_IMAGE}

RUN printf '%s\n' '#!/usr/bin/env bash' 'echo elyro-example-tool' >/usr/local/bin/elyro-example-tool \
  && chmod +x /usr/local/bin/elyro-example-tool
