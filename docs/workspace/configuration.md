# Workspace Configuration

Elyro needs no project file when one built-in Toolchain can be detected. Add `elyro.yaml` at the project root only when the project needs named Environments, a custom image, a fixed platform, ports, mounts, privileged access, or editor settings.

```yaml
version: 1
default_environment: dev
environments:
  dev:
    toolchain: go
    platform: linux/arm64
    docker:
      publish:
        - "8080:8000"
      mounts:
        - source: .cache
          target: /home/elyro/.cache/example
          read_only: false
    vscode:
      extensions:
        - golang.go
      settings:
        go.toolsManagement.autoUpdate: false
```

Unknown fields and unsupported configuration versions are errors. `elyro doctor --project-dir .` validates the file and prints the resolved Environment, Toolchain, image, platform, and Workspace state without changing the project or Workspace.

## Selection and precedence

- `elyro up --environment NAME` selects a named Environment and requires `elyro.yaml`.
- `elyro up --toolchain python|go|node|java` selects a built-in Toolchain directly and bypasses the configured default Environment.
- `--environment` and `--toolchain` are mutually exclusive.
- Without either flag, Elyro uses `default_environment` when it is set; otherwise it detects exactly one Toolchain from project markers.
- `--platform` overrides the configured platform.
- Repeated `--publish` values merge with `docker.publish`; conflicting mappings for the same host port are rejected.
- Changing the resolved Environment, image reference, platform, ports, mounts, privileged mode, or hostname causes `elyro up` to recreate the Workspace. Use `elyro up --recreate` when a custom image was rebuilt under the same tag.

## Environment fields

Each entry below `environments` must set `toolchain`, `image`, or both. A project-owned derived image also sets `build`:

```yaml
version: 1
default_environment: api
environments:
  api:
    toolchain: go
  custom:
    image: example/my-workspace:local
  custom-go:
    toolchain: go
    image: example/my-go-workspace:local
  derived:
    toolchain: go
    image: elyro-local/example:dev
    build:
      context: .
      dockerfile: .elyro/Dockerfile
```

`toolchain` selects one of Elyro's maintained images and recommended editor settings. `image` selects a project-owned image that satisfies the [custom image contract](custom-images.md). When both are present, the custom image is used and the Toolchain contributes metadata and editor recommendations.

`build` declares a project-owned Docker build. It must be paired with `image`; both `context` and `dockerfile` are non-empty paths relative to the project and must remain inside the project after symlinks are resolved. The target image must have an explicit tag other than `latest` and must not use a digest. Elyro only executes this build through `elyro image build`; `elyro up` never builds it implicitly and never automatically pulls an explicitly configured image.

Use `elyro image init` to add this block and a commented `.elyro/Dockerfile` safely. Existing Environments, fields, ordering, and YAML comments are preserved. See [Project-derived and fully custom images](custom-images.md) for ownership and runtime requirements.

`platform` accepts `linux/amd64` or `linux/arm64`. When omitted, Elyro uses the host architecture.

## Docker options

Application ports bind to host loopback only:

```yaml
docker:
  publish:
    - "8000"
    - "18080:8080"
```

Mount sources may be absolute, start with `~/`, or be relative to the project root. Targets must be absolute Linux paths:

```yaml
docker:
  mounts:
    - source: fixtures
      target: /opt/project-fixtures
      read_only: true
```

Mounting a host path outside the project, mounting `/var/run/docker.sock` or `/run/docker.sock`, or enabling `privileged: true` requires an explicit `elyro up --allow-unsafe-environment`. Elyro reports every unsafe reason before creating or replacing a container.

## VS Code and Cursor

`vscode.extensions` and `vscode.settings` are written to the generated Workspace configuration used by both VS Code and Cursor Remote SSH:

```yaml
vscode:
  extensions:
    - ms-python.python
  settings:
    python.defaultInterpreterPath: /usr/bin/python3
```

Built-in Toolchains supply recommended settings and extensions. Project entries extend the extension list and override settings with the same key.
