# Validation Matrix

| Changed scope | Minimum | Behavior or integration |
| :--- | :--- | :--- |
| Go under `cmd/`, `internal/`, or `skills/` | gofmt, owning tests, `go test ./...`, `go vet ./...` | `make workspace-smoke` or `make workspace-e2e` |
| Workspace lifecycle, JSON, SSH, editor, examples | relevant Go tests | `make workspace-smoke`, `make workspace-e2e` |
| Embedded Skill | skill validator, isolated-HOME command tests | installed-binary show/install/uninstall smoke |
| Workspace images | matching build and smoke | four-image amd64/arm64 Nightly |
| Make, scripts, or workflows | syntax, ShellCheck error level, affected target | matching CI, Nightly, or Release workflow |
| Release configuration | `make release-config-check`, installer and Formula smoke | tagged Release and post-release artifact checks |
| Documentation | changed relative links and renamed-term scan | run copied commands when behavior changed |

Always run `git diff --check`, preserve unrelated work, and keep credentials and machine-specific state out of the repository.
