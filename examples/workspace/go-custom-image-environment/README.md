# Go Custom Image Environment

This example shows how to run a project-defined workspace environment backed by a business image derived from `elyro/workspace-go:dev-amd64`.

## Build The Custom Image

From the repository root:

```bash
make workspace-go-image-build
docker build --platform linux/amd64 \
  -t elyro/examples/go-custom-environment:dev \
  examples/workspace/go-custom-image-environment
```

To build an arm64 variant instead:

```bash
make workspace-go-image-build WORKSPACE_GO_PLATFORM=linux/arm64 WORKSPACE_BASE_PLATFORM=linux/arm64 RUNTIME_BASE_PLATFORM=linux/arm64
docker build --platform linux/arm64 \
  --build-arg WORKSPACE_GO_IMAGE=elyro/workspace-go:dev-arm64 \
  -t elyro/examples/go-custom-environment:dev \
  examples/workspace/go-custom-image-environment
```

## Start The Workspace

From this example directory:

```bash
elyro up --environment api --publish 8000
```

Elyro reads `elyro.yaml`, resolves the `api` environment, keeps Go-oriented VS Code defaults, and starts the custom image declared by the environment.

To run an arm64 environment explicitly:

```bash
elyro up --environment api --platform linux/arm64 --publish 8000
```

## Verify

Inside the workspace shell:

```bash
elyro-example-tool
go run .
```

Then reach the service from the host at `http://127.0.0.1:8000/healthz`.
