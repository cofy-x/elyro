# Runtime environment example

This public example demonstrates explicit, reproducible variables for the Workspace container:

```bash
cp -R examples/workspace/go-runtime-environment /tmp/elyro-runtime-environment
cd /tmp/elyro-runtime-environment
elyro up
elyro exec -- go run .
elyro doctor --json
elyro down
```

`.elyro/dev.env` provides shared non-sensitive defaults. `docker.environment` overrides files, so `APP_MODE` is `inline-wins`. To add machine-local values, copy `.elyro/user.local.env.example` to `.elyro/user.local.env`, add that path after `dev.env` in `docker.env_files`, and keep the local file ignored by Git.

These values are runtime configuration, not secrets. Anyone with Docker access can read them from the container configuration. Use a dedicated secret manager when a value must remain secret.
