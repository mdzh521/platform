# Backend Config

`backend-center` now supports a single YAML config file plus environment-variable overrides.

## Load Order

1. Built-in defaults
2. Config file
3. Environment variables

Environment variables always win.

## Default Config File

The default file is:

`backend-center/config/platform-center.yaml`

You can also point to another file with:

`APP_CONFIG_FILE=/app/config/platform-center.yaml`

If `APP_CONFIG_FILE` is not set, the backend will try:

1. `config/platform-center.yaml`
2. `platform-center.yaml`
3. `config.yaml`

The shipped default config keeps `frontend.mode: embedded` so the backend can still serve the bundled UI when run standalone.
If you deploy with a separate frontend container, override it with `FRONTEND_MODE=external`.

## Main Fields

- `app_name`: service name shown in runtime info
- `port`: backend listen port
- `run_mode`: current run mode, default `single`
- `jwt_secret`: JWT signing secret
- `internal_runner_token`: internal bearer token used by `terraform-runner`
- `default_admin_username`: initial admin username
- `default_admin_password`: initial admin password
- `mysql_*`: MySQL connection info
- `cloud_accounts`: optional bootstrap cloud accounts written into the cloud account table at startup
- `frontend.mode`: `embedded` or `external`
- `frontend.api_base_url`: frontend API base URL, usually empty in same-origin deployments
- `cache.driver`: `memory` or `redis`
- `cache.redis_url`: Redis DSN for cache when driver is `redis`
- `queue.driver`: `memory` or `redis`
- `queue.redis_url`: Redis DSN for queue when driver is `redis`

## Cloud Credentials

If you want cloud credentials to be file-configurable instead of entering them in the UI, edit:

`backend-center/config/platform-center.yaml`

Example:

```yaml
cloud_accounts:
  - name: aws-demo
    provider: aws
    access_key: AKIA...
    secret_key: your-secret-key
    region: ap-southeast-1
    role_arn: arn:aws:iam::123456789012:role/platform-center
    default_tags:
      - env=dev
      - owner=platform
    default_zones:
      - ap-southeast-1a
      - ap-southeast-1b
```

Notes:

- These credentials are encrypted before being stored in the database.
- Updating the YAML and restarting the backend will upsert the account by `name`.
- If you leave `access_key` or `secret_key` empty for an existing account, the old encrypted value is kept.
- For container deployment, edit the repo file and rebuild/restart `backend`.

## Production Advice

- Do not keep `jwt_secret: change-me`
- Do not keep `internal_runner_token: runner-dev-token`
- Do not keep the default admin password
- Prefer env overrides for secrets in production
- Keep non-sensitive defaults in the YAML file
