FROM ghcr.io/cofy-x/elyro/workspace-go:v0.1.3

RUN apt-get update \
    && apt-get install -y --no-install-recommends sqlite3 \
    && rm -rf /var/lib/apt/lists/*
