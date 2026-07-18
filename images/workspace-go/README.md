# Workspace Go

`elyro/workspace-go:dev-amd64` / `elyro/workspace-go:dev-arm64` are local Elyro Workspace toolchain images for Go development.

## What It Provides

- Built on top of `elyro/workspace-base:dev-<arch>` in source builds
- a pinned, checksummed Go toolchain
- CGO development build dependencies

Project-level linters and formatters are intentionally not installed. Pin them in the project or provide a custom image when they are part of the project's development contract.

## Build

From the repository root:

```bash
make workspace-go-image-build
make workspace-go-image-build-arm64
```

The selected platform is also tagged explicitly, for example `elyro/workspace-go:dev-arm64`.
The Go archive uses its upstream official download URL by default. Controlled development environments may explicitly route the download through the shared `ELYRO_PROXY_URL` / `ELYRO_PROXY_NO_PROXY` settings.

## Use

```bash
make elyro-build
bin/elyro up --toolchain go
```
