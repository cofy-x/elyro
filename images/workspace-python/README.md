# Workspace Python

`elyro/workspace-python:dev-amd64` / `elyro/workspace-python:dev-arm64` are local Elyro Workspace toolchain images for Python development.

## What It Provides

- Built on top of `elyro/workspace-base:dev-<arch>` in source builds
- `python3`, `python3-dev`, `python3-venv`, and development build dependencies
- a pinned, checksummed standalone `uv`
- `python -> python3`

This image intentionally does not preinstall vendor-specific SDKs or any sensitive environment variables.

## Build

From the repository root:

```bash
make workspace-python-image-build
make workspace-python-image-build-arm64
```

The selected platform is also tagged explicitly, for example `elyro/workspace-python:dev-arm64`.
The build uses the shared `ELYRO_PROXY_URL` / `ELYRO_PROXY_NO_PROXY` settings for
the standalone `uv` release download and does not add a global Python
package-management layer.

## Use

```bash
make elyro-build
bin/elyro up --toolchain python

# Inside the Workspace
uv venv
uv sync
uv run python app.py
python -m venv .venv-standard
```
