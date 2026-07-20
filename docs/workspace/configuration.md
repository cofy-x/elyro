# Workspace Configuration

Elyro needs no project file when one built-in Toolchain can be detected. Add `elyro.yaml` at the project root only when the project needs named Environments, a custom image, a fixed platform, ports, mounts, runtime variables, privileged access, or editor settings.

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
      environment:
        APP_ENV: development
      env_files:
        - .elyro/dev.env
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
- Changing the resolved Environment, image reference, platform, ports, mounts, effective runtime environment, privileged mode, or hostname causes `elyro up` to recreate the Workspace. Use `elyro up --dry-run` to inspect the exact reasons first. Use `elyro up --recreate` when a custom image was rebuilt under the same tag.

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

### Runtime environment

Use `docker.environment` for explicit values and `docker.env_files` for project-relative files:

```yaml
docker:
  environment:
    APP_ENV: development
    CGO_ENABLED: "1"
  env_files:
    - .elyro/dev.env
    - .elyro/user.local.env
```

Every key and value in `environment` must be a YAML string; quote values such as `"1"`, `"true"`, and the empty string. Names must match `[A-Za-z_][A-Za-z0-9_]*`. Values cannot contain NUL or newlines.

Environment-file paths must be non-empty, unique, relative to the project, and resolve to regular files inside it after symlinks are followed. Files are UTF-8 and accept LF or CRLF, blank lines, comment lines, and `KEY=VALUE`. The first `=` separates the name and value; empty values and `#` within a value are preserved. Bare keys, duplicate keys within one file, `export KEY=...`, and invalid names fail validation. Quotes and `${VAR}` are literal text—Elyro performs no interpolation.

Image `ENV` has the lowest priority. Files override it in configuration order, and `docker.environment` overrides every file. Elyro computes Workspace identity from the final effective values, so changing an overridden source does not recreate the Workspace while changing an effective value does.

Elyro does not read `.env`, inherit host variables, or provide `up -e`/`--env-file`. For a one-command override, use `elyro exec -- env KEY=value command`. Runtime values are passed into Docker container configuration and can be inspected by anyone with Docker access; they are not secrets. Keep local files in `.gitignore` and use a secret manager for sensitive material. See the [runtime environment example](../../examples/workspace/go-runtime-environment/README.md).

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
