# Releasing Elyro

A release tag must point to the exact commit that passed Main CI and the manual amd64/arm64 Nightly matrix.

The current release surface contains:

- four archives for macOS/Linux amd64/arm64, each containing only `elyro`;
- `checksums.txt`, archive SBOMs, provenance attestations, and a GitHub Release whose prerelease status is derived from the semantic-version tag;
- four signed multi-architecture images: `workspace-base`, `workspace-python`, `workspace-go`, and `workspace-node`;
- a generated Homebrew Formula that installs only `elyro`.

Before tagging, run:

```bash
go test ./...
go vet ./...
make image-report-test
make release-config-check
make release-install-smoke
make workspace-smoke
make workspace-e2e
make ci-pr-smoke
git diff --check
```

Repository maintainers dispatch Nightly and create the annotated tag through the private Hangar remote-sync workflow. A Mac development checkout must not push the branch or tag directly.

A manually dispatched release Nightly publishes unique candidate tags in the form `candidate-<full-commit-sha>-<run-id>`. It first validates both native architectures and Workspace behavior, then creates four multi-architecture indexes and checks their OCI revision labels, compressed-size budgets from `release/image-budgets.json`, and deltas from `ELYRO_COMPARE_VERSION`. Existing candidate tags are never overwritten; rerun a failed candidate as a new workflow dispatch.

Scheduled Nightly runs build and test both architectures without publishing candidates. The tag-triggered Release workflow repeats the manifest, label, comparison, and budget checks against the final versioned images.

Ordinary tags such as `v0.1.5` publish a non-prerelease GitHub Release and become latest. A future suffix such as `v0.2.0-rc.1` is marked prerelease automatically and does not replace the latest production release.
