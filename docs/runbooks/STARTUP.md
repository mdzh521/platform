# Platform Center Startup

## Quick Start

```bash
cd /Users/alex/ops/platform-center
docker compose -f ops/compose/docker-compose.yml up -d --build
```

启动后访问：

- 前端：`http://127.0.0.1:8080/`
- 后端 API：`http://127.0.0.1:18080/`
- terraform-runner health：`http://127.0.0.1:18091/healthz`
- resource-sync-worker health：`http://127.0.0.1:18092/healthz`
- machine-enrollment-worker health：`http://127.0.0.1:18093/healthz`
- cluster-enrollment-worker health：`http://127.0.0.1:18094/healthz`

默认管理员：

- 用户名：`admin`
- 密码：`admin123456`

## 启动前最该看的配置

- 后端：
  [platform-center.yaml](/Users/alex/ops/platform-center/apps/control-plane/config/platform-center.yaml)
- Terraform Runner：
  [terraform-runner.yaml](/Users/alex/ops/platform-center/apps/terraform-runner/config/terraform-runner.yaml)
- Resource Sync Worker：
  [resource-sync-worker.yaml](/Users/alex/ops/platform-center/apps/resource-sync-worker/config/resource-sync-worker.yaml)
- Machine Enrollment Worker：
  [machine-enrollment-worker.yaml](/Users/alex/ops/platform-center/apps/machine-enrollment-worker/config/machine-enrollment-worker.yaml)
- Cluster Enrollment Worker：
  [cluster-enrollment-worker.yaml](/Users/alex/ops/platform-center/apps/cluster-enrollment-worker/config/cluster-enrollment-worker.yaml)

统一说明见：

- [DEPLOYMENT_CONFIG.md](/Users/alex/ops/platform-center/docs/runbooks/DEPLOYMENT_CONFIG.md)
- [PHASE4_ENROLLMENT_RUNTIME.md](/Users/alex/ops/platform-center/docs/runbooks/PHASE4_ENROLLMENT_RUNTIME.md)

## 验证

最小 smoke：

```bash
zsh /Users/alex/ops/platform-center/ops/scripts/smoke_check.sh
```

真实 provider enrollment 前置检查：

```bash
zsh /Users/alex/ops/platform-center/ops/scripts/check_enrollment_runtime.sh
```
