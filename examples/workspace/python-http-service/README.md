# Python HTTP Service

This is a generic Python service example for Elyro Workspace. It is intentionally simple and does not depend on any company-specific packages, private registries, or sensitive environment variables.

## Start Elyro Workspace

From this example directory:

```bash
elyro up --toolchain python --publish 8000
```

Then open VS Code and use `Remote-SSH: Connect to Host...`, choose the generated `workspace-python-http-service` host, and open the mounted remote project folder (for this example, `/home/elyro/python-http-service`).

## Initialize the Environment

Inside the workspace shell:

```bash
uv --version
python --version
python -m venv .venv
. .venv/bin/activate
uv pip install -e .
```

You can also use:

```bash
uv sync
. .venv/bin/activate
```

## Configure

```bash
cp .env.example .env.local
export APP_HOST=0.0.0.0
export APP_PORT=8000
export APP_GREETING="hello from Elyro Workspace"
```

## Run the Service

```bash
python app.py
```

In another terminal on the host:

```bash
curl http://127.0.0.1:8000/healthz
curl http://127.0.0.1:8000/
```

Expected responses:

```text
{"status":"ok"}
{"message":"hello from Elyro Workspace"}
```
