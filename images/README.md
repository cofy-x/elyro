# Elyro Workspace Images

Elyro publishes five images for `linux/amd64` and `linux/arm64`:

- `workspace-base`: Ubuntu, UID/GID 1000 user `elyro`, SSH, sudo, shell, Git, curl, jq, and process tools.
- `workspace-python`: Python, `venv`, build dependencies, and checksummed standalone `uv`.
- `workspace-go`: checksummed Go Toolchain and CGO build dependencies.
- `workspace-node`: checksummed Node.js LTS, npm, Corepack, and native-addon build dependencies.
- `workspace-java`: JDK, Maven, and Gradle.

The base image is built directly from a digest-pinned Ubuntu 24.04 image. It contains no language Toolchain and no coding-agent CLI.

Official Toolchain images contain the language runtime, its primary build or dependency-management tools, and the native compilation prerequisites needed for ordinary project work. Optional project tools such as linters and formatters are not part of the image contract; projects should pin them themselves or use a custom image.

## Build

```bash
make workspace-base-image-build
make workspace-python-image-build
make workspace-go-image-build
make workspace-node-image-build
make workspace-java-image-build
```

Append `-arm64` to a target for a local arm64 build. Source builds use architecture-specific tags such as `elyro/workspace-go:dev-arm64`; public releases publish exact multi-architecture tags such as `ghcr.io/cofy-x/elyro/workspace-go:v0.1.0`.

Official sources and an empty proxy are the defaults. Controlled development environments may explicitly set `ELYRO_PROXY_URL`, `ELYRO_MIRROR_SOURCE`, and `ELYRO_GOPROXY`. `ELYRO_PROXY_URL=auto` is opt-in; Elyro never probes a local proxy by default.

## Report published size

```bash
make image-report VERSION=v0.1.0
make image-report VERSION=v0.2.0 COMPARE_VERSION=v0.1.0 FORMAT=json
make image-report VERSION=v0.1.0 TOP_LAYERS=10
```

The report reads remote OCI manifests without pulling images and requires both supported architectures. Compressed size is the sum of platform layer descriptor sizes. Versions must be exact `v`-prefixed tags.

Maintainer candidate and Release workflows additionally enforce the reviewed limits in `release/image-budgets.json`. Candidate-only report flags are workflow implementation details; ordinary users should continue reporting immutable version tags with the commands above.

Projects may instead provide a fully owned image that satisfies the [custom image contract](../docs/workspace/custom-images.md).
