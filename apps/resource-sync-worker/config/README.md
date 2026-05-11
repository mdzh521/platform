# Resource Sync Worker Config

`resource-sync-worker` now supports:

1. built-in defaults
2. YAML config file
3. environment-variable overrides

Environment variables always win.

## Default File

`resource-sync-worker/config/resource-sync-worker.yaml`

You can also point to another file with:

`SYNC_WORKER_CONFIG_FILE=/app/config/resource-sync-worker.yaml`

If `SYNC_WORKER_CONFIG_FILE` is not set, the worker will try:

1. `config/resource-sync-worker.yaml`
2. `resource-sync-worker.yaml`
3. `config.yaml`

## Main Fields

- `worker_name`: worker identity reported to backend
- `listen_addr`: health endpoint listen address
- `poll_interval`: backend sync polling interval
- `backend_base_url`: backend internal API base URL
- `backend_token`: backend internal bearer token
