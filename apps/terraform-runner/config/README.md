# Terraform Runner Config

`terraform-runner` now supports:

1. built-in defaults
2. YAML config file
3. environment-variable overrides

Environment variables always win.

## Default File

`terraform-runner/config/terraform-runner.yaml`

You can also point to another file with:

`RUNNER_CONFIG_FILE=/app/config/terraform-runner.yaml`

If `RUNNER_CONFIG_FILE` is not set, the runner will try:

1. `config/terraform-runner.yaml`
2. `terraform-runner.yaml`
3. `config.yaml`

## Main Fields

- `runner_name`: worker identity reported to backend
- `listen_addr`: health endpoint listen address
- `poll_interval`: claim polling interval
- `heartbeat_interval`: execution heartbeat interval
- `backend_base_url`: backend internal API base URL
- `backend_token`: backend internal bearer token
- `workspace_root`: job workspace root
- `provider_cache_dir`: Terraform provider cache directory
- `template_root`: local template root
- `max_parallel_jobs`: runner parallelism
- `workspace_success_ttl`: cleanup TTL for planned/succeeded workspaces
- `workspace_failure_ttl`: cleanup TTL for failed/cancelled workspaces
- `workspace_destroyed_ttl`: cleanup TTL for destroyed workspaces
