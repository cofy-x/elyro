# Local CI Orchestration

- `pr-smoke.sh`: Go tests and Workspace smoke.
- `nightly.sh`: PR baseline plus Workspace e2e.
- `weekly.sh`: the complete nightly suite.
- `run-suite.sh`: dispatches `pr`, `nightly`, `weekly`, or `all`.
- `check-release-inputs.sh`: verifies exact release inputs and four-image definitions.
- `build-core-images.sh` / `smoke-core-images.sh`: builds and validates all four Workspace images for one explicit platform.
- `push-candidate-images.sh` / `merge-candidate-images.sh`: publishes unique per-architecture candidate images and combines them into immutable dual-architecture indexes during an authorized manual Nightly.
- `candidate-images-test.sh`: validates candidate tag immutability, per-architecture push, index merge, and invalid-input failures without registry writes.
- `install-release-smoke.sh`: verifies checksum selection and a one-binary archive.
- `homebrew-formula-smoke.sh`: verifies Formula generation and optional real macOS installation.

```bash
scripts/ci/run-suite.sh pr
scripts/ci/run-suite.sh nightly
scripts/ci/check-release-inputs.sh v0.1.6
make release-install-smoke
```
