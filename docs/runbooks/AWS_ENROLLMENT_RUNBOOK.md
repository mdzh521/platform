# AWS Enrollment Runbook

日期：2026-04-26

## 当前结论

这份 runbook 只覆盖 `EKS enrollment`。

当前状态：

- `aws-sdk` 是默认主路径
- `aws-cli` 只作为 fallback
- `docker compose` 运行面已对齐当前 schema 和 internal API
- 但 compose 运行面还没有完成真实 AWS provider 验证

所以当前推荐结论是：

- `P0 真实取证优先使用 host process 跑 cluster-enrollment-worker`
- `compose worker` 目前可以用于 schema / internal API 对齐验证
- 在没有真实 AWS 凭据与 live EKS 资源验证前，不把 compose 方案写成正式闭环

## Adapter 路径

EKS enrollment 的实际调用顺序：

1. `aws-sdk`
2. `aws-cli` fallback

默认目标字段：

- `endpoint`
- `version`
- `vpc`
- `subnet refs`
- `provider status`

## 推荐运行位置

### P0 真实取证

推荐：

- 在宿主机直接运行 `cluster-enrollment-worker`

原因：

- 最容易复用宿主机现成 `~/.aws`、profile、SSO 或临时凭据
- 更容易确认 `aws-sdk` 和 `aws-cli` 到底各自是否可用
- 排查 provider auth、网络、profile 绑定时更直接

### Compose 运行面

当前用途：

- 验证 backend schema 是否与仓库代码一致
- 验证 internal claim/result API 是否连通
- 验证 worker 容器是否能正常启动

当前不宣称：

- compose 已完成真实 AWS provider 闭环

## 推荐凭据来源

优先顺序：

1. 环境变量
2. `~/.aws/credentials`
3. `~/.aws/config`
4. AWS profile / SSO 派生配置

最小环境变量：

```bash
export AWS_REGION=ap-southeast-1
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
```

如果使用临时凭据，再加：

```bash
export AWS_SESSION_TOKEN=...
```

## 推荐启动命令

### 1. 检查 runtime

```bash
cd /Users/alex/ops/platform-center
zsh scripts/check_enrollment_runtime.sh
```

### 2. 启动 backend/runtime

```bash
cd /Users/alex/ops/platform-center
docker compose up -d backend resource-sync-worker terraform-runner
```

如果只验证 schema / worker 容器，也可以加：

```bash
docker compose up -d machine-enrollment-worker cluster-enrollment-worker
```

### 3. 推荐 host process 跑 cluster worker

```bash
cd /Users/alex/ops/platform-center/apps/cluster-enrollment-worker
BACKEND_API_BASE_URL=http://127.0.0.1:18080 \
BACKEND_API_TOKEN=runner-dev-token \
AWS_REGION=ap-southeast-1 \
go run ./cmd/cluster-enrollment-worker
```

## Phase5 取证命令

真实 EKS apply 完成后，至少回报：

1. `source job id`
2. `cloud resource id`
3. `cluster-enrollment claim/result`
4. `final k8s cluster record`

取证入口：

```bash
cd /Users/alex/ops/platform-center
zsh scripts/collect_enrollment_evidence.sh <source_job_id> [cloud_resource_id]
```

## 当前已确认的 runtime/schema 基线

当前本地 compose 已确认：

- `cloud_resources.cluster_enrollment_status` 存在
- `cloud_resources.cluster_enrollment_worker` 存在
- `cloud_resources.cluster_enrollment_error` 存在
- `clusters.source_resource_id` 存在

这表示：

- 当前 backend/db 已具备 phase4/phase5 所需的 schema 取证能力

## 故障排查

### backend 启不来

先看：

```bash
docker compose logs --tail=100 backend
```

如果是路由或 schema 类问题，先修代码并重新：

```bash
docker compose up -d --build backend resource-sync-worker machine-enrollment-worker cluster-enrollment-worker terraform-runner
```

### worker 没 claim 到任务

先看：

```bash
docker compose logs --tail=100 cluster-enrollment-worker
docker compose logs --tail=100 backend
```

然后确认：

- `cloud_resources` 里是否存在 `resource_role=cluster`
- `lifecycle_state=managed`
- `cluster_enrollment_status in (pending, failed)`

### provider 调用失败

先区分是：

- `aws-sdk` 失败
- `aws-cli` fallback 失败

当前判断方法：

- 看 worker 输出错误
- 看 `cluster_enrollment_error`
- 看 `collect_enrollment_evidence.sh` 结果

### 没有最终 managed cluster record

优先检查：

1. `cloud resource` 是否真的进入 cluster enrollment
2. `cluster_enrollment_status` 是否为 `enrolled` / `access_pending`
3. `cloud_id` 是否可映射到 `clusters.source_resource_id`

## 当前边界

当前还没有完成的，不要误报为完成：

- 第一条真实 EKS provider 证据链
- compose 运行面真实 provider 闭环
- ACK 真实链路
