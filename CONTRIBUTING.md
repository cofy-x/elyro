# Contributing to Elyro

Elyro welcomes issues and pull requests that improve its local development
runtime workflows.

## Development setup

Requirements: Go, Docker with Compose v2, an SSH client, and GNU Make.

```bash
make elyro-build
make test
make release-config-check
make release-install-smoke
make workspace-smoke
make workspace-e2e
```

Changes to release configuration additionally require GoReleaser `v2.13.3`,
the version pinned by GitHub Actions. Homebrew installation smoke runs in CI;
maintainers on macOS can reproduce it with
`ELYRO_HOMEBREW_INSTALL_SMOKE=1 scripts/ci/homebrew-formula-smoke.sh`.

Coding agents should read `AGENTS.md`, then use the `.x/README.md` task router instead of loading every repository context document. Add tests for behavior changes and update the affected user documentation.

## Pull requests

- Keep each pull request focused and explain user-visible behavior.
- Do not commit credentials, generated databases, local state, or proprietary
  third-party binaries.
- Use signed-off commits only when required by your organization; Elyro does
  not currently require a separate contributor license agreement.
- By submitting a contribution, you agree that it is licensed under
  Apache-2.0.
