# Security Policy

## Supported versions

Elyro is pre-1.0 software. Security fixes are applied to the latest published
minor release and the default branch.

## Reporting a vulnerability

Do not open a public issue for suspected vulnerabilities. Use GitHub's private
security advisory flow for `cofy-x/elyro`. Include affected versions, impact,
reproduction steps, and any proposed mitigation.

Elyro manages local containers, host mounts, credentials, SSH configuration,
and browser terminals. Treat project-provided `elyro.yaml` files and custom
images as executable configuration, and review unsafe-Environment warnings before
proceeding.

Values configured through `docker.environment` or `docker.env_files` are stored in Docker container configuration and are visible to users with Docker access. They are runtime configuration, not a secret store. Do not place credentials or other sensitive values there unless that Docker visibility is acceptable.
