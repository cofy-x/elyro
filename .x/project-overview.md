# Repository Ownership Map

Elyro has one product domain: a local Linux Workspace for a host project. Human and coding-agent callers use the same CLI contract.

| Path | Ownership |
| :--- | :--- |
| `cmd/elyro` | Top-level CLI wiring, doctor, version, and init commands |
| `internal/cliui` | Shared human-facing terminal styles, capability detection, receipts, and next-step presentation |
| `internal/workspace` | Configuration, Toolchain detection, registry, lifecycle, Docker execution, SSH/editor handoff, JSON behavior, and Skill command behavior |
| `internal/images` | Development and release image reference resolution |
| `skills/use-elyro-workspace` | Canonical host coding-agent Skill and Codex UI metadata |
| `images/workspace-*` | Four supported Workspace images |
| `mk/`, `scripts/`, `.github/workflows/`, `release/` | Build, validation, packaging, and release orchestration |
| `docs/` | User-facing product and maintenance documentation |
| `.x/` | Agent-facing repository routing and validation context |

## Workspace invariants

- Only explicit initialization commands write project configuration: `elyro init` creates `elyro.yaml`, while `elyro image init` adds the project image build and Dockerfile. Zero-config `up` never creates or builds project files.
- `shell` and `exec` use local `docker exec` as user `elyro` in the project mount. SSH exists only for editor handoff and direct user access.
- Container identity stays path-hashed; hostname is the sanitized project name and participates in specification comparison.
- SSH config and known-hosts records are Elyro-managed and isolated from the user's global trust records.
- JSON schema changes require contract tests. JSON stdout contains only the document; progress and actionable errors use stderr.
- Human presentation may use restrained semantic color only on a TTY. Piped output, `NO_COLOR`, `TERM=dumb`, and CI stay plain; `exec`, `shell`, and `skill show` remain transparent streams.
- The embedded Skill contains guidance only: no scripts, credentials, model configuration, or direct Docker/SSH instructions.

## Structural sync points

- Command changes update the CLI contract test, README, CLI reference, and Skill when its workflow changes.
- Image changes update the four-image release matrix, image report, smoke scripts, release inputs, and image docs.
- Packaging changes update GoReleaser, installer, Homebrew generation/smoke, release workflow, and releasing docs.
- Path moves update this map, `.x/README.md`, Make/script consumers, and documentation links in the same change.
