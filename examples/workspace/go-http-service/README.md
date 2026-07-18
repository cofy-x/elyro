# Go HTTP Service

This is a generic Go service example for Elyro Workspace. It is intentionally simple and does not depend on private modules or sensitive environment variables.

## Start Elyro Workspace

From this example directory:

```bash
elyro up --toolchain go --publish 8000
```

Then open VS Code and use `Remote-SSH: Connect to Host...`, choose the generated `workspace-go-http-service` host, and open the mounted remote project folder (for this example, `/home/elyro/go-http-service`).

## Run the Service

Inside the workspace shell:

```bash
go version
go test ./...
go run .
```

On the host (with `--publish 8000`):

```bash
curl http://127.0.0.1:8000/healthz
curl http://127.0.0.1:8000/
```

Expected responses:

```text
{"status":"ok"}
{"message":"hello from Elyro Workspace"}
```

Optional:

```bash
export APP_GREETING="custom greeting"
go run .
```
