# control-plane

`apps/control-plane` is the backend API and embedded-web runtime for `platform-center`.

## Current Scope

- system domain
  - auth
  - user
  - rbac
  - identity source
  - system settings
- k8s domain
  - clusters
  - namespaces
  - workloads
  - resource explorer
  - interactive terminal
- machine domain
  - asset groups
  - machine assets
  - credential library
  - SSH terminal
  - SFTP
  - sessions and events

## Startup

Prefer using the repo root compose entry:

```bash
cd /Users/alex/ops/platform-center
docker compose -f ops/compose/docker-compose.yml up -d --build
```

See:

- [Startup Guide](../../docs/runbooks/STARTUP.md)
- [Deployment Config](../../docs/runbooks/DEPLOYMENT_CONFIG.md)
- [Backend Config](./config/README.md)
- [Smoke Check Script](../../ops/scripts/smoke_check.sh)

## Config

Backend supports:

1. built-in defaults
2. YAML config file
3. environment variable overrides

Default file:

`apps/control-plane/config/platform-center.yaml`

## Domain Boundary

The codebase is still one repo, but should be treated as a modular monolith.

Shared infrastructure capability belongs in:

- `internal/infra/*`

Business domains belong in:

- `internal/domains/iam/*`
- `internal/domains/platform/*`
- `internal/domains/delivery/*`
- `internal/domains/delivery/summary`
- `internal/domains/delivery/blueprints`
- `internal/domains/delivery/cloud`
- `internal/domains/clusters/*`
- `internal/domains/machines/*`
- `internal/domains/projects/*`

See:

- [System Domain Boundary](../../docs/architecture/system-domain-boundary.md)
